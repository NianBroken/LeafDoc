// 本文件统一 Cookie 操作的安全连接判断，供认证与 CSRF Cookie 共用。
package utils

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

// 不符合安全条件且未允许明文连接时，调用入口据此拒绝 Cookie 操作。
var ErrHTTPSRequired = errors.New("https is required for secure cookies")

// RequireSecure 优先识别直接 TLS，再识别转发协议及两个代理安全标记。
// X-Forwarded-Proto 使用小写后的包含判断，不严格拆分协议列表；
// 代理头是否可信不由本函数验证，应由部署和上游处理链保证。
// 允许明文时返回 false 与空错误，否则返回 HTTPS 必需错误。
func RequireSecure(c *gin.Context, allowInsecure bool) (bool, error) {
	if c.Request.TLS != nil {
		return true, nil
	}
	xfp := strings.ToLower(c.GetHeader("X-Forwarded-Proto"))
	if strings.Contains(xfp, "https") {
		return true, nil
	}

	if strings.EqualFold(c.GetHeader("X-Forwarded-Ssl"), "on") {
		return true, nil
	}

	if strings.EqualFold(c.GetHeader("Front-End-Https"), "on") {
		return true, nil
	}
	if allowInsecure {
		return false, nil
	}
	return false, ErrHTTPSRequired
}
