// 本文件统一 Cookie 安全属性判断，实际 TLS 连接是唯一协议依据。
package utils

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// ErrHTTPSRequired 保留旧调用接口的错误标识。
var ErrHTTPSRequired = errors.New("https is required for secure cookies")

// RequireSecure 同时支持明文与加密连接，兼容参数不改变协议识别。
// 网关通过真实 TLS 传递 HTTPS 状态，不信任客户端协议头。
func RequireSecure(c *gin.Context, allowInsecure bool) (bool, error) {
	return c.Request.TLS != nil, nil
}
