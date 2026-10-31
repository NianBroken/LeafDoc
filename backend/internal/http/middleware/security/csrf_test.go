package security

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestCSRFMiddleware_AllowsSafeMethodsWithoutToken 以 GET 代表免令牌路径，确认请求到达处理函数。
// 用例只发出 GET，不将其断言范围扩大到 HEAD 和 OPTIONS。
func TestCSRFMiddleware_AllowsSafeMethodsWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.Use(CSRFMiddleware(csrf))
	router.GET("/safe", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/safe", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET without CSRF, got %d", w.Code)
	}
}

// TestCSRFMiddleware_BlocksPostWithoutCookie 确认安全连接上的 POST 缺少 Cookie 时返回缺失提示。
func TestCSRFMiddleware_BlocksPostWithoutCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.Use(CSRFMiddleware(csrf))
	router.POST("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/protected", nil)
	req.TLS = &tls.ConnectionState{} // 设置安全连接标记，使拒绝原因限定为缺少 Cookie。
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for POST without CSRF cookie, got %d", w.Code)
	}

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "CSRF token missing" {
		t.Fatalf("expected error 'CSRF token missing', got '%s'", body["error"])
	}
}

// TestCSRFMiddleware_BlocksPostWithCookieButNoHeader 确认 Cookie 本身不足以通过双重提交检查。
func TestCSRFMiddleware_BlocksPostWithCookieButNoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.Use(CSRFMiddleware(csrf))
	router.POST("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/protected", nil)
	req.TLS = &tls.ConnectionState{}
	// 只附带 Cookie，不提供请求头或表单令牌，预期返回无效令牌提示。
	req.AddCookie(&http.Cookie{
		Name:  "__Host-leafdoc_csrf",
		Value: "test-token",
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for POST with cookie but no header, got %d", w.Code)
	}

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "Invalid CSRF token" {
		t.Fatalf("expected error 'Invalid CSRF token', got '%s'", body["error"])
	}
}

// TestCSRFMiddleware_BlocksPostWithMismatchingTokens 提供两个不同值，验证比较失败的响应。
func TestCSRFMiddleware_BlocksPostWithMismatchingTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.Use(CSRFMiddleware(csrf))
	router.POST("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := strings.NewReader("")
	req := httptest.NewRequest("POST", "/protected", body)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{
		Name:  "__Host-leafdoc_csrf",
		Value: "cookie-token",
	})
	req.Header.Set("X-CSRF-Token", "different-header-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for POST with mismatching tokens, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Invalid CSRF token" {
		t.Fatalf("expected error 'Invalid CSRF token', got '%s'", resp["error"])
	}
}

// TestCSRFMiddleware_AllowsPostWithMatchingTokens 使用相同 Cookie 与请求头，确认 POST 被放行。
func TestCSRFMiddleware_AllowsPostWithMatchingTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.Use(CSRFMiddleware(csrf))
	router.POST("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	const token = "same-token"
	body := strings.NewReader("")
	req := httptest.NewRequest("POST", "/protected", body)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{
		Name:  "__Host-leafdoc_csrf",
		Value: token,
	})
	req.Header.Set("X-CSRF-Token", token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for POST with valid CSRF, got %d", w.Code)
	}
}

// TestCSRFMiddleware_SkipsPostWhenAPIKeyResolved 由上游模拟已解析密钥标记，验证免令牌路径。
// 此处不测试密钥数据库解析，只检查 CSRF 对上下文标记的响应。
func TestCSRFMiddleware_SkipsPostWhenAPIKeyResolved(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(APIKeyAuthContextKey, true)
		c.Next()
	})
	router.Use(CSRFMiddleware(csrf))
	router.POST("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/protected", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for POST with a resolved API key and no CSRF token, got %d", w.Code)
	}
}

// TestCSRFMiddleware_DoesNotSkipForBearerHeaderAlone 确认仅构造 Bearer 头不能获得密钥豁免。
func TestCSRFMiddleware_DoesNotSkipForBearerHeaderAlone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.Use(CSRFMiddleware(csrf))
	router.POST("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/protected", nil)
	req.TLS = &tls.ConnectionState{}
	req.Header.Set("Authorization", "Bearer lw_abcdefgh_secret")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403: a Bearer header that was not resolved to a key must not skip CSRF, got %d", w.Code)
	}
}

// TestCSRFMiddleware_DoesNotSkipWhenAPIKeyFlagIsFalse 确认标记必须等于 true 而非只要求键存在。
func TestCSRFMiddleware_DoesNotSkipWhenAPIKeyFlagIsFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(APIKeyAuthContextKey, false)
		c.Next()
	})
	router.Use(CSRFMiddleware(csrf))
	router.POST("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/protected", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 when the API key flag is false, got %d", w.Code)
	}
}
