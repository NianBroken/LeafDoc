// 本文件验证 Cookie 协议判断不接受客户端伪造的代理头。
package utils

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRequireSecure 验证实际 TLS 状态优先且普通 HTTP 默认可用。
func TestRequireSecure(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		response := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(response)
		c.Request = httptest.NewRequest("GET", "http://localhost/", nil)
		c.Request.Header.Set("X-Forwarded-Proto", "https")
		c.Request.Header.Set("X-Forwarded-Ssl", "on")
		c.Request.Header.Set("Front-End-Https", "on")
		if encrypted {
			c.Request.TLS = &tls.ConnectionState{}
		}
		secure, err := RequireSecure(c, false)
		if err != nil || secure != encrypted {
			t.Fatalf("TLS=%v secure=%v error=%v", encrypted, secure, err)
		}
	}
}
