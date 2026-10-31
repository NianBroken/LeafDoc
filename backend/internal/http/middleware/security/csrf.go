package security

import (
	"crypto/subtle"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthContextKey 保存密钥鉴权中间件成功解析用户后设置的布尔标记。
// 常量定义在安全包，使导入安全包的鉴权包能够共享名称而不形成循环引用。
const APIKeyAuthContextKey = "apiKeyAuth"

// CSRFMiddleware 校验请求显式提交的令牌是否与 Cookie 中的令牌一致。
// GET、HEAD、OPTIONS 直接通过，其余方法除已解析的 API 密钥请求外均需校验。
// 缺少 Cookie、读取失败、缺少提交值或值不一致都以 403 终止处理链。
func CSRFMiddleware(csrf *CSRFCookie) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		// 只按这三个明确列出的读取或预检方法放行，不将其他方法归为免检查。
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}

		// 放行依据是服务解析成功后写入的 true 标记，单独携带请求头不能绕过检查。
		// 该路径使用显式提交的密钥凭据，不要求浏览器 Cookie 的双重提交令牌。
		if resolved, _ := c.Get(APIKeyAuthContextKey); resolved == true {
			c.Next()
			return
		}

		// 读取过程也检查连接安全要求，读取错误与空值对外使用同一种缺失提示。
		cookieToken, err := csrf.Read(c)
		if err != nil || cookieToken == "" {
			slog.Default().Warn("CSRF token missing or error reading token", "error", err)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "CSRF token missing",
			})
			return
		}

		// 优先使用 X-CSRF-Token 请求头，仅在请求头为空时读取表单字段。
		headerToken := c.GetHeader("X-CSRF-Token")
		if headerToken == "" {
			headerToken = c.PostForm("csrf_token")
		}

		// 对非空值使用标准库比较函数核对字节，长度不同或内容不同均拒绝。
		if headerToken == "" || subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookieToken)) != 1 {
			slog.Default().Warn("CSRF token invalid or does not match cookie")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Invalid CSRF token",
			})
			return
		}

		c.Next()
	}
}
