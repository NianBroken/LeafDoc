// 安装会话按顺序校验配置，敏感字段只保存在服务端内存。
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/installation"
	"golang.org/x/crypto/bcrypt"
)

const setupSessionLifetime = 30 * time.Minute
const setupCookieName = "leafdoc_setup"

type setupSession struct {
	Config     installation.Config
	Step       int
	Revision   int
	Expires    time.Time
	Inspection database.Inspection
	Format     bool
}

type setupStepRequest struct {
	installation.Config
	SetupToken      string `json:"setupToken"`
	Revision        int    `json:"revision"`
	Step            int    `json:"step"`
	ConfirmDatabase string `json:"confirmDatabase"`
}

// installationSteps 包装最终安装处理器，阻止旧式一次提交绕过逐步校验。
func installationSteps(next http.Handler, base, token string) http.Handler {
	var mu sync.Mutex
	sessions := map[string]*setupSession{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, base)
		if path != "/api/setup" && !strings.HasPrefix(path, "/api/setup/") || path == "/api/setup/ca" {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		// 同源标记不可由普通跨站表单设置，Cookie 不作为唯一请求授权依据。
		if r.Method != http.MethodGet && r.Header.Get("X-LeafDoc-Setup") != "1" {
			setupJSON(w, 403, map[string]any{"message": "请求来源无效，请刷新安装页面"})
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for id, s := range sessions {
			if time.Now().After(s.Expires) {
				delete(sessions, id)
			}
		}
		var session *setupSession
		if cookie, err := r.Cookie(setupCookieName); err == nil {
			session = sessions[cookie.Value]
		}
		if r.Method == http.MethodGet && path == "/api/setup/status" {
			if session == nil {
				setupJSON(w, 200, map[string]any{"step": 0, "revision": 0})
				return
			}
			setupJSON(w, 200, setupSessionView(session))
			return
		}
		if r.Method != http.MethodPost {
			setupJSON(w, 405, map[string]any{"message": "不支持的方法"})
			return
		}
		var input setupStepRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			setupJSON(w, 400, map[string]any{"message": "请求格式无效"})
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			setupJSON(w, 400, map[string]any{"message": "请求只能包含一个 JSON 对象"})
			return
		}
		if path == "/api/setup/token" {
			if subtle.ConstantTimeCompare([]byte(input.SetupToken), []byte(token)) != 1 {
				setupJSON(w, 403, map[string]any{"message": "安装令牌无效，请核对实例的 setup-token 文件", "state": "failed"})
				return
			}
			if len(sessions) >= 64 {
				setupJSON(w, 503, map[string]any{"message": "安装会话过多，请稍后重试", "state": "blocked"})
				return
			}
			random := make([]byte, 32)
			if _, err := rand.Read(random); err != nil {
				setupJSON(w, 500, map[string]any{"message": "无法创建安装会话，请重试"})
				return
			}
			session = &setupSession{Step: 1, Revision: 1, Expires: time.Now().Add(setupSessionLifetime)}
			sessions[hex.EncodeToString(random)] = session
			cookiePath := base + "/"
			http.SetCookie(w, &http.Cookie{Name: setupCookieName, Value: hex.EncodeToString(random), Path: cookiePath, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: int(setupSessionLifetime.Seconds())})
			setupJSON(w, 200, setupSessionView(session))
			return
		}
		if session == nil {
			setupJSON(w, 401, map[string]any{"message": "安装会话已过期，请重新验证安装令牌", "state": "failed"})
			return
		}
		if input.Revision != session.Revision {
			setupJSON(w, 409, map[string]any{"message": "步骤状态已变化，请刷新后重试", "state": "failed"})
			return
		}
		session.Expires = time.Now().Add(setupSessionLifetime)
		if path == "/api/setup/back" {
			if input.Step < 1 || input.Step >= session.Step {
				setupJSON(w, 400, map[string]any{"message": "不能跳过未完成的步骤"})
				return
			}
			session.Step = input.Step
			session.Revision++
			session.Format = false
			if input.Step <= 3 {
				session.Inspection = database.Inspection{}
			}
			setupJSON(w, 200, setupSessionView(session))
			return
		}
		if path == "/api/setup" {
			if session.Step != 5 {
				setupJSON(w, 409, map[string]any{"message": "请先完成全部步骤校验"})
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
			defer cancel()
			if err := database.PrepareSetup(ctx, session.Config, session.Inspection, session.Format); err != nil {
				session.Step = 3
				session.Format = false
				session.Revision++
				setupFailure(w, session, err)
				return
			}
			payload := struct {
				installation.Config
				SetupToken string `json:"setupToken"`
			}{session.Config, token}
			body, err := json.Marshal(payload)
			if err != nil {
				setupFailure(w, session, err)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
			r.Header.Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
			return
		}
		expected := map[string]int{"/api/setup/address": 1, "/api/setup/credentials": 2, "/api/setup/database": 3, "/api/setup/admin": 4}
		step, ok := expected[path]
		if !ok || session.Step != step {
			setupJSON(w, 409, map[string]any{"message": "请按顺序完成当前步骤"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		c := session.Config
		var err error
		switch step {
		case 1:
			c.Host = strings.TrimSpace(input.Host)
			c.Port = input.Port
			if c.Host == "" || c.Port < 1 || c.Port > 65535 || strings.ContainsAny(c.Host, "/\\ \t\r\n") {
				err = errors.New("请填写有效的数据库主机和 1 至 65535 的端口")
				break
			}
			var connection net.Conn
			connection, err = (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
			if err == nil {
				err = connection.Close()
			}
		case 2:
			c.User = input.User
			c.Password = input.Password
			c.TLS = input.TLS
			c.CA = input.CA
			var db *sql.DB
			db, err = database.ConnectSetup(ctx, c, false)
			if err == nil {
				err = db.Close()
			}
		case 3:
			c.Database = strings.TrimSpace(input.Database)
			var checked database.Inspection
			checked, err = database.InspectSetup(ctx, c)
			if err == nil {
				if input.ConfirmDatabase != "" && (session.Config.Database != c.Database || session.Inspection.Fingerprint != checked.Fingerprint) {
					session.Inspection = checked
					setupFailure(w, session, errors.New("数据库结构已变化，请重新检查后再确认清空"))
					return
				}
				session.Config = c
				session.Inspection = checked
				session.Format = false
				if checked.State == "incompatible" && input.ConfirmDatabase != c.Database {
					setupJSON(w, 200, map[string]any{"step": 3, "revision": session.Revision, "inspection": checked, "state": "confirmation", "message": "数据库结构不兼容。拒绝清空会保留全部数据，当前无法继续安装。"})
					return
				}
				session.Format = checked.State == "incompatible"
			}
		case 4:
			c.Admin = strings.TrimSpace(input.Admin)
			c.Email = strings.TrimSpace(input.Email)
			c.AdminPassword = input.AdminPassword
			if c.Admin == "" || len(c.AdminPassword) < 8 || len(c.AdminPassword) > 72 {
				err = errors.New("管理员账号不能为空，密码须为 8 至 72 字节")
				break
			}
			if session.Inspection.State == "compatible" {
				err = verifySetupAdmin(ctx, &c)
			} else {
				_, err = mail.ParseAddress(c.Email)
			}
		}
		if err != nil {
			setupFailure(w, session, err)
			return
		}
		session.Config = c
		session.Step++
		session.Revision++
		setupJSON(w, 200, setupSessionView(session))
	})
}

// verifySetupAdmin 使用只读连接验证已有管理员，不在验证阶段触发建表。
func verifySetupAdmin(ctx context.Context, c *installation.Config) (err error) {
	db, err := database.ConnectSetup(ctx, *c, true)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	var username, password, email string
	err = db.QueryRowContext(ctx, "SELECT username,password,email FROM users WHERE role='admin' LIMIT 1").Scan(&username, &password, &email)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = mail.ParseAddress(c.Email)
		return err
	}
	if err != nil {
		return err
	}
	if username != c.Admin || bcrypt.CompareHashAndPassword([]byte(password), []byte(c.AdminPassword)) != nil {
		return errors.New("已有实例需要原管理员账号和密码")
	}
	c.Email = email
	return nil
}

func setupSessionView(s *setupSession) map[string]any {
	c := s.Config
	c.Password = ""
	c.AdminPassword = ""
	c.Secret = ""
	c.CA = ""
	return map[string]any{"step": s.Step, "revision": s.Revision, "config": c, "inspection": s.Inspection, "format": s.Format, "state": "passed"}
}

func setupFailure(w http.ResponseWriter, s *setupSession, err error) {
	message := database.SetupError(err)
	for _, secret := range []string{s.Config.Password, s.Config.AdminPassword, s.Config.Secret} {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[已隐藏]")
		}
	}
	state := "failed"
	var network net.Error
	if errors.As(err, &network) || errors.Is(err, context.DeadlineExceeded) {
		state = "blocked"
		message = "暂时无法连接数据库服务，请检查地址、端口、网络与服务状态。" + message
	}
	setupJSON(w, 400, map[string]any{"message": message, "step": s.Step, "revision": s.Revision, "state": state})
}

func setupJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	w.WriteHeader(status)
	writeInstallationResponse(w, string(body))
}
