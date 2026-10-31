package security

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NianBroken/LeafDoc/internal/http/middleware/utils"
	"github.com/gin-gonic/gin"
)

// TestCSRFCookie_CookieName_Secure 确认安全模式使用带主机约束前缀的 Cookie 名称。
func TestCSRFCookie_CookieName_Secure(t *testing.T) {
	csrf := NewCSRFCookie(false, time.Hour)

	name := csrf.cookieName(true)
	if name != "__Host-leafdoc_csrf" {
		t.Errorf("Expected secure CSRF cookie name '__Host-leafdoc_csrf', got '%s'", name)
	}
}

// TestCSRFCookie_CookieName_Insecure 确认普通模式使用独立名称，不复用安全 Cookie。
func TestCSRFCookie_CookieName_Insecure(t *testing.T) {
	csrf := NewCSRFCookie(true, time.Hour)

	name := csrf.cookieName(false)
	if name != "leafdoc_csrf" {
		t.Errorf("Expected insecure CSRF cookie name 'leafdoc_csrf', got '%s'", name)
	}
}

// TestCSRFCookie_Issue_Secure 用带 TLS 状态的请求触发签发，检查完整 Cookie 属性与响应头。
// 这里使用内存路由，不建立真实 TLS 连接，TLS 状态只供安全策略识别。
func TestCSRFCookie_Issue_Secure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		_, err := csrf.Issue(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// 首次请求应只写一个 Cookie，检查可被脚本读取、根路径和一小时有效期。
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	csrfCookie := cookies[0]
	if csrfCookie.Name != "__Host-leafdoc_csrf" {
		t.Errorf("Expected CSRF cookie name '__Host-leafdoc_csrf', got '%s'", csrfCookie.Name)
	}
	if csrfCookie.Value == "" {
		t.Error("Expected CSRF cookie to have a non-empty value")
	}
	if csrfCookie.HttpOnly {
		t.Error("Expected CSRF cookie to NOT be HttpOnly")
	}
	if !csrfCookie.Secure {
		t.Error("Expected CSRF cookie to be Secure in secure mode")
	}
	if csrfCookie.Path != "/" {
		t.Errorf("Expected CSRF cookie path '/', got '%s'", csrfCookie.Path)
	}
	if csrfCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("Expected CSRF cookie SameSite LaxMode, got %v", csrfCookie.SameSite)
	}
	if csrfCookie.MaxAge != int(time.Hour.Seconds()) {
		t.Errorf("Expected CSRF cookie MaxAge %d, got %d", int(time.Hour.Seconds()), csrfCookie.MaxAge)
	}

	// 响应头与 Cookie 必须携带同一个非空值，便于前端选择读取入口。
	headerToken := w.Result().Header.Get("X-CSRF-Token")
	if headerToken == "" {
		t.Error("Expected X-CSRF-Token header to be set")
	}
	if headerToken != csrfCookie.Value {
		t.Errorf("Expected header token '%s' to match cookie value '%s'", headerToken, csrfCookie.Value)
	}
}

// TestCSRFCookie_Issue_Insecure 验证明确允许普通连接时签发不带 Secure 属性的令牌。
func TestCSRFCookie_Issue_Insecure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := NewCSRFCookie(true, time.Hour)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		_, err := csrf.Issue(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	csrfCookie := cookies[0]
	if csrfCookie.Name != "leafdoc_csrf" {
		t.Errorf("Expected CSRF cookie name 'leafdoc_csrf', got '%s'", csrfCookie.Name)
	}
	if csrfCookie.Secure {
		t.Error("Expected CSRF cookie to NOT be Secure in insecure mode")
	}
	if csrfCookie.HttpOnly {
		t.Error("Expected CSRF cookie to NOT be HttpOnly")
	}
}

// TestCSRFCookie_Issue_ErrorWhenHTTPSRequired 将安全策略错误映射为 400，并确认普通请求被拒绝。
func TestCSRFCookie_Issue_ErrorWhenHTTPSRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		_, err := csrf.Issue(c)
		if err != nil {
			if err == utils.ErrHTTPSRequired {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	// 请求不设置 TLS 或 HTTPS 转发标记，且策略禁止普通连接。
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 when HTTPS required for CSRF cookie, got %d", w.Code)
	}
}

// TestCSRFCookie_Read_Secure 通过安全名称传入固定令牌，确认读取路径返回成功状态。
// 本用例断言状态码，不解析响应 JSON 来额外核对返回文本。
func TestCSRFCookie_Read_Secure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		token, err := csrf.Read(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{
		Name:  "__Host-leafdoc_csrf",
		Value: "test-csrf-token",
	})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// TestCSRFCookie_Read_Insecure 使用普通模式名称，确认允许普通连接的读取请求成功。
func TestCSRFCookie_Read_Insecure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := NewCSRFCookie(true, time.Hour)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		token, err := csrf.Read(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "leafdoc_csrf",
		Value: "test-csrf-token-insecure",
	})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// TestCSRFCookie_Read_MissingCookie 保持连接安全条件成立，只验证缺少 Cookie 的错误分支。
func TestCSRFCookie_Read_MissingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		token, err := csrf.Read(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 when CSRF cookie is missing, got %d", w.Code)
	}
}

// TestCSRFCookie_Clear_Secure 确认安全模式的删除响应使用正确名称、空值与负有效期。
func TestCSRFCookie_Clear_Secure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := NewCSRFCookie(false, time.Hour)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		err := csrf.Clear(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	csrfCookie := cookies[0]
	if csrfCookie.Name != "__Host-leafdoc_csrf" {
		t.Errorf("Expected CSRF cookie name '__Host-leafdoc_csrf', got '%s'", csrfCookie.Name)
	}
	if csrfCookie.Value != "" {
		t.Errorf("Expected CSRF cookie value to be empty, got '%s'", csrfCookie.Value)
	}
	if csrfCookie.MaxAge != -1 {
		t.Errorf("Expected CSRF cookie MaxAge -1, got %d", csrfCookie.MaxAge)
	}
}

// TestCSRFCookie_Clear_Insecure 确认普通模式只删除普通名称对应的令牌 Cookie。
func TestCSRFCookie_Clear_Insecure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := NewCSRFCookie(true, time.Hour)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		err := csrf.Clear(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	csrfCookie := cookies[0]
	if csrfCookie.Name != "leafdoc_csrf" {
		t.Errorf("Expected CSRF cookie name 'leafdoc_csrf', got '%s'", csrfCookie.Name)
	}
	if csrfCookie.MaxAge != -1 {
		t.Errorf("Expected CSRF cookie MaxAge -1, got %d", csrfCookie.MaxAge)
	}
}
