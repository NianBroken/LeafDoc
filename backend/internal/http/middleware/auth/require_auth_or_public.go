// 本文件为公开读取路由复用统一身份解析，在未认证时读取运行中的公开开关。
package auth

import (
	"net/http"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/publicaccess"
	"github.com/gin-gonic/gin"
)

// RequireAuthOrPublicRead 保留成功解析的用户并执行强制改密检查。
// 只有未认证状态且公开开关非空并已启用时匿名放行，服务器错误仍拒绝。
// 开关在每个请求中读取，本函数不检查 HTTP 方法，路由装配负责只接入允许公开的读取。
func RequireAuthOrPublicRead(authService *auth.AuthService, authCookies *AuthCookies, public publicaccess.ReadGate) gin.HandlerFunc {
	return func(c *gin.Context) {
		r := resolveSession(c, authService, authCookies)
		if rejectPasswordChange(c, r.user) {
			return
		}
		if r.user != nil {
			c.Next()
			return
		}
		if r.status == http.StatusUnauthorized && public != nil && public.Enabled() {
			c.Next() // 公开读取已启用，允许无用户身份继续。
			return
		}
		c.AbortWithStatusJSON(r.status, r.body)
	}
}
