// 本文件使用内存 HTTP 路由验证匿名公开读取与已注入身份的分流规则。
// 配置缺失与公开开关切换分别断言，不连接账号数据库或签发真实 JWT。
package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/publicaccess"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// newAuthOrPublicRouter 在可选身份注入之后挂载公开读取鉴权，响应区分用户与匿名状态。
// 测试处理函数只接受真实 User 指针，便于检查中间件上下文契约。
func newAuthOrPublicRouter(auth *coreauth.AuthService, authCookies *authmw.AuthCookies, gate publicaccess.ReadGate, inject gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if inject != nil {
		r.Use(inject)
	}
	r.Use(authmw.RequireAuthOrPublicRead(auth, authCookies, gate))
	r.GET("/test", func(c *gin.Context) {
		if v, ok := c.Get("user"); ok {
			c.JSON(http.StatusOK, gin.H{"user": v.(*coreauth.User).Username})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": nil})
	})
	return r
}

// doGet 发出不带 TLS 标记的内存 GET 请求，只在参数非空时附带指定 Cookie。
func doGet(r http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestRequireAuthOrPublicRead_UserAlreadyInContext_Passes 确认上游用户可在关闭公开访问时直接读取。
func TestRequireAuthOrPublicRead_UserAlreadyInContext_Passes(t *testing.T) {
	authCookies := authmw.NewAuthCookies(true, time.Hour, 24*time.Hour)
	inject := func(c *gin.Context) {
		c.Set("user", &coreauth.User{ID: "x", Username: "proxied", Role: coreauth.RoleEditor})
		c.Next()
	}
	r := newAuthOrPublicRouter(nil, authCookies, &flipGate{enabled: false}, inject)

	w := doGet(r, nil)
	if w.Code != http.StatusOK || w.Body.String() != `{"user":"proxied"}` {
		t.Fatalf("expected 200 with proxied user, got %d %s", w.Code, w.Body.String())
	}
}

// TestRequireAuthOrPublicRead_NoToken_PublicOff_Returns401 核对未登录私有访问的状态与完整错误体。
func TestRequireAuthOrPublicRead_NoToken_PublicOff_Returns401(t *testing.T) {
	authCookies := authmw.NewAuthCookies(true, time.Hour, 24*time.Hour)
	r := newAuthOrPublicRouter(nil, authCookies, &flipGate{enabled: false}, nil)

	w := doGet(r, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if w.Body.String() != `{"error":"Missing or invalid access token"}` {
		t.Fatalf("expected RequireAuth's no-token body, got %s", w.Body.String())
	}
}

// TestRequireAuthOrPublicRead_NoToken_PublicOn_PassesAnonymously 确认公开读取不虚构用户身份。
func TestRequireAuthOrPublicRead_NoToken_PublicOn_PassesAnonymously(t *testing.T) {
	authCookies := authmw.NewAuthCookies(true, time.Hour, 24*time.Hour)
	r := newAuthOrPublicRouter(nil, authCookies, &flipGate{enabled: true}, nil)

	w := doGet(r, nil)
	if w.Code != http.StatusOK || w.Body.String() != `{"user":null}` {
		t.Fatalf("expected 200 anonymous, got %d %s", w.Code, w.Body.String())
	}
}

// TestRequireAuthOrPublicRead_NilAuthServiceWithToken_Returns500 确认服务缺失错误不会降级为匿名成功。
func TestRequireAuthOrPublicRead_NilAuthServiceWithToken_Returns500(t *testing.T) {
	authCookies := authmw.NewAuthCookies(true, time.Hour, 24*time.Hour)
	// 提供非空令牌迫使请求进入验证路径，即使公开读取开启也返回配置错误。
	r := newAuthOrPublicRouter(nil, authCookies, &flipGate{enabled: true}, nil)

	w := doGet(r, &http.Cookie{Name: "leafdoc_at", Value: "something"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d %s", w.Code, w.Body.String())
	}
}

// TestRequireAuthOrPublicRead_NilGate_TreatedAsPublicOff 确认缺少公开读取门禁时采用私有访问规则。
func TestRequireAuthOrPublicRead_NilGate_TreatedAsPublicOff(t *testing.T) {
	authCookies := authmw.NewAuthCookies(true, time.Hour, 24*time.Hour)
	r := newAuthOrPublicRouter(nil, authCookies, nil, nil)

	w := doGet(r, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for nil gate, got %d", w.Code)
	}
}

// TestRequireAuthOrPublicRead_GateFlipsAtRuntime_SameRouterChangesBehaviour 在同一路由上切换门禁。
// 每次请求重新读取门禁值，连续验证关闭、开启和再次关闭时的访问结果。
func TestRequireAuthOrPublicRead_GateFlipsAtRuntime_SameRouterChangesBehaviour(t *testing.T) {
	authCookies := authmw.NewAuthCookies(true, time.Hour, 24*time.Hour)
	gate := &flipGate{enabled: false}
	r := newAuthOrPublicRouter(nil, authCookies, gate, nil)

	if w := doGet(r, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("public off: expected 401, got %d", w.Code)
	}

	gate.enabled = true
	if w := doGet(r, nil); w.Code != http.StatusOK {
		t.Fatalf("public on: expected 200 on the same router, got %d", w.Code)
	}

	gate.enabled = false
	if w := doGet(r, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("public off again: expected 401 on the same router, got %d", w.Code)
	}
}

// flipGate 提供可直接切换的测试门禁，不参与并发访问。
type flipGate struct{ enabled bool }

// Enabled 返回当前字段而非构造时快照，使同一个路由能够观察后续设置。
func (g *flipGate) Enabled() bool { return g.enabled }
