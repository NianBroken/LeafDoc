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
	"github.com/NianBroken/LeafDoc/internal/core/dualhttp"
	"github.com/NianBroken/LeafDoc/internal/core/installation"
	"github.com/NianBroken/LeafDoc/internal/core/shared"
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
	// 安装引导与业务服务共用证书和同端口双协议识别。
	if cert, key := os.Getenv("LEAFDOC_TLS_CERT"), os.Getenv("LEAFDOC_TLS_KEY"); cert != "" || key != "" {
		combined, wrapErr := dualhttp.Wrap(listener, cert, key)
		if wrapErr != nil {
			return errors.Join(wrapErr, listener.Close())
		}
		listener = combined
	}
	defer shared.LogNetworkClose(listener, "close installation listener")
	completed := make(chan *installation.Config, 1)
	var mu sync.Mutex
	done := false
	base := normalizeBasePath(cfg.server.basePath)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		path := strings.TrimPrefix(r.URL.Path, base)
		if path == "/api/setup/ca" && r.Method == http.MethodGet {
			serveInstallationCA(w, r, os.Getenv("LEAFDOC_SETUP_CA_FILE"))
			return
		}
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
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
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
				http.Error(w, "数据库结构或读写权限校验失败: "+database.SetupError(probeErr), http.StatusBadRequest)
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
				http.Error(w, "数据库连接或初始化失败: "+database.SetupError(err), http.StatusBadRequest)
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
	server := &http.Server{Handler: installationSteps(handler, base, token), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second}
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
