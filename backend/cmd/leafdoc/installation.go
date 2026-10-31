// 安装入口先提供独立引导服务，完整构造 Wiki 并保存配置后才交给正式启动流程。
package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/installation"
	"github.com/NianBroken/LeafDoc/internal/wiki"
	"golang.org/x/crypto/bcrypt"
)

// prepareInstallation 仅对显式启用网页安装的实例等待配置；旧开发流程保留原有启动约定。
func prepareInstallation(ctx context.Context, cfg *serverConfig) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	c, err := installation.Load(cfg.server.dataDir)
	if err == nil {
		cfg.auth.jwtSecret = c.Secret
		cfg.auth.adminUsername = c.Admin
		cfg.auth.adminEmail = c.Email
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("安装配置不可用: %w", err)
	}
	if os.Getenv("LEAFDOC_WEB_SETUP") != "true" {
		return nil
	}
	tokenPath := filepath.Join(cfg.server.dataDir, "setup-token")
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		return fmt.Errorf("读取安装令牌: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if len(token) < 32 {
		return errors.New("安装令牌长度不足")
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(cfg.server.host, cfg.server.port))
	if err != nil {
		return err
	}
	completed := make(chan *installation.Config, 1)
	var mu sync.Mutex
	done := false
	base := normalizeBasePath(cfg.server.basePath)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		path := strings.TrimPrefix(r.URL.Path, base)
		if path == "/api/health" || path == "/api/setup/status" {
			w.Header().Set("Content-Type", "application/json")
			writeInstallationResponse(w, `{"initialized":false,"phase":"setup"}`)
			return
		}
		if path == "/api/setup" && r.Method == http.MethodPost {
			mu.Lock()
			defer mu.Unlock()
			if done {
				http.Error(w, "初始化已完成", http.StatusConflict)
				return
			}
			if r.Header.Get("Content-Type") != "application/json" {
				http.Error(w, "需要 JSON 请求", http.StatusUnsupportedMediaType)
				return
			}
			var request struct {
				installation.Config
				SetupToken string `json:"setupToken"`
			}
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				http.Error(w, "配置格式无效", http.StatusBadRequest)
				return
			}
			if subtle.ConstantTimeCompare([]byte(request.SetupToken), []byte(token)) != 1 {
				http.Error(w, "安装令牌无效", http.StatusForbidden)
				return
			}
			candidate := request.Config
			if _, err := candidate.MySQL(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(candidate.Admin) == "" || len(candidate.AdminPassword) < 8 {
				http.Error(w, "管理员账号不能为空，密码至少八个字符", http.StatusBadRequest)
				return
			}
			if _, err := mail.ParseAddress(candidate.Email); err != nil {
				http.Error(w, "管理员邮箱无效", http.StatusBadRequest)
				return
			}
			secret := make([]byte, 64)
			if _, err := rand.Read(secret); err != nil {
				http.Error(w, "生成密钥失败", http.StatusInternalServerError)
				return
			}
			candidate.Secret = hex.EncodeToString(secret)
			candidate.Complete = false
			installation.Candidate(cfg.server.dataDir, &candidate)
			defer installation.Candidate(cfg.server.dataDir, nil)

			probeCtx, cancelProbe := context.WithTimeout(r.Context(), 30*time.Second)
			probeErr := database.VerifyWritable(probeCtx, cfg.server.dataDir)
			cancelProbe()
			if probeErr != nil {
				http.Error(w, "数据库结构或读写权限校验失败", http.StatusBadRequest)
				return
			}

			// 已有数据库需要证明管理员凭据，不能通过换一套安装资料接管旧实例。
			store, openErr := coreauth.NewUserStore(cfg.server.dataDir)
			if openErr != nil {
				http.Error(w, "数据库连接或权限校验失败", http.StatusBadRequest)
				return
			}
			existing, lookupErr := store.GetAdminUser()
			closeErr := store.Close()
			if closeErr != nil {
				http.Error(w, "关闭数据库校验资源失败", http.StatusInternalServerError)
				return
			}
			if lookupErr == nil {
				if existing.Username != candidate.Admin || bcrypt.CompareHashAndPassword([]byte(existing.Password), []byte(candidate.AdminPassword)) != nil {
					http.Error(w, "已有实例需要原管理员账号和密码", http.StatusForbidden)
					return
				}
				candidate.Email = existing.Email
			} else if !errors.Is(lookupErr, coreauth.ErrUserNotFound) {
				http.Error(w, "读取既有管理员失败", http.StatusBadRequest)
				return
			}

			// Wiki 构造复用全部真实表结构与管理员校验，失败时不写安装成功标记。
			instance, err := wiki.NewWiki(&wiki.WikiOptions{StorageDir: cfg.server.dataDir, AdminUsername: candidate.Admin, AdminEmail: candidate.Email, AdminPassword: candidate.AdminPassword, JWTSecret: candidate.Secret, AccessTokenTimeout: cfg.auth.accessTokenTimeout, RefreshTokenTimeout: cfg.auth.refreshTokenTimeout, EnableRevision: cfg.frontend.enableRevision, EnableAPIKeyManagement: cfg.frontend.enableAPIKeyManagement, MaxRevisionHistory: cfg.frontend.maxRevisionHistory, RevisionCoalesceWindow: cfg.frontend.revisionCoalesceWindow})
			if err != nil {
				_ = database.ClosePools()
				http.Error(w, "数据库连接或初始化失败，请检查地址、凭据、权限和服务器状态", http.StatusBadRequest)
				return
			}
			startupErr := instance.WaitForStartup()
			if err = errors.Join(startupErr, instance.Close(), database.ClosePools()); err != nil {
				http.Error(w, "搜索初始化或关闭初始化资源失败", http.StatusInternalServerError)
				return
			}
			candidate.Complete = true
			candidate.AdminPassword = ""
			if err = installation.Save(cfg.server.dataDir, &candidate); err != nil {
				http.Error(w, "配置保存失败，请检查数据目录权限和磁盘空间", http.StatusInternalServerError)
				return
			}
			done = true
			w.Header().Set("Content-Type", "application/json")
			writeInstallationResponse(w, `{"initialized":true}`)
			completed <- &candidate
			return
		}
		if strings.HasPrefix(path, "/api/") {
			http.Error(w, "请先完成初始化", http.StatusServiceUnavailable)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "不支持的方法", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		encodedBase, _ := json.Marshal(base)
		writeInstallationResponse(w, strings.ReplaceAll(installationPage, "__BASE_PATH__", string(encodedBase)))
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second}
	stopped := make(chan error, 1)
	go func() { stopped <- server.Serve(listener) }()
	var result error
	select {
	case c = <-completed:
		cfg.auth.jwtSecret = c.Secret
		cfg.auth.adminUsername = c.Admin
		cfg.auth.adminEmail = c.Email
	case <-ctx.Done():
		result = ctx.Err()
	case err = <-stopped:
		result = err
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return errors.Join(result, server.Shutdown(shutdownCtx))
}

// writeInstallationResponse 写出引导页或状态正文，连接写入失败保留到服务日志。
func writeInstallationResponse(w http.ResponseWriter, body string) {
	if _, err := fmt.Fprint(w, body); err != nil {
		slog.Error("安装响应写入失败", "error", err)
	}
}

// installationPage 自包含样式与表单使安装不依赖数据库、外部字体或第三方网络资源。
const installationPage = `<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>LeafDoc 首次启动初始化</title><style>body{font:16px system-ui;background:#f4f5f7;color:#182125;margin:0}main{max-width:560px;margin:5vh auto;padding:32px;background:white;border-radius:12px}label{display:block;margin:16px 0}input,textarea,button{box-sizing:border-box;width:100%;padding:10px;margin-top:6px;font:inherit}button{background:#17634d;color:white;border:0;cursor:pointer}#message{white-space:pre-wrap}input[type=checkbox]{width:auto}</style><main><h1>LeafDoc 首次启动初始化</h1><p>连接外部 MySQL，配置将保存在本安装实例中。数据库需预先创建并授权。</p><form id="setup"><label>安装令牌<input name="token" type="password" required autocomplete="off"></label><label>数据库地址<input name="host" required></label><label>数据库端口<input name="port" type="number" value="3306" min="1" max="65535" required></label><label>数据库名<input name="database" required></label><label>数据库账号<input name="user" required></label><label>数据库密码<input name="password" type="password" autocomplete="new-password"></label><label><input name="tls" type="checkbox"> 使用 TLS 验证数据库证书</label><label>自定义 CA 证书<textarea name="ca" placeholder="可留空以使用系统 CA"></textarea></label><label>管理员账号<input name="admin" required></label><label>管理员邮箱<input name="email" type="email" required></label><label>管理员密码<input name="adminPassword" type="password" minlength="8" required autocomplete="new-password"></label><button>验证并完成初始化</button></form><p id="message" role="status"></p></main><script>const f=document.querySelector('form'),m=document.querySelector('#message');f.onsubmit=async e=>{e.preventDefault();const b=f.querySelector('button');b.disabled=true;m.textContent='正在验证数据库并准备系统，请等待';try{const v=Object.fromEntries(new FormData(f)),token=v.token;delete v.token;v.setupToken=token;v.port=Number(v.port);v.tls=f.elements.tls.checked;const base=__BASE_PATH__;const r=await fetch(base+'/api/setup',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(v)});if(!r.ok)throw Error(await r.text());m.textContent='初始化完成，正在进入系统';setTimeout(()=>location.assign(base+'/'),1500)}catch(e){m.textContent=e.message;b.disabled=false}};</script></html>`
