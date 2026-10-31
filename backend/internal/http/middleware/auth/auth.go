// 本文件统一从上下文或访问 Cookie 解析身份，再提供严格认证、可选认证与角色门禁。
// 解析结果和响应写入分开，公开读取入口可按同一失败结果选择匿名放行。
package auth

import (
	"net/http"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/gin-gonic/gin"
)

const errUserNotAuthenticated = "User not authenticated"

// sessionOutcome 保存成功用户或拒绝状态与正文，不在解析过程中直接写入响应。
type sessionOutcome struct {
	user   *auth.User
	status int   // 用户非空时保持零值，失败时保存 HTTP 状态码。
	body   gin.H // 成功时为空，失败时供门禁写入统一字符串错误。
}

// denied 创建失败结果，保留指定状态与消息，不调用 Gin 中止入口。
func denied(status int, message string) sessionOutcome {
	return sessionOutcome{status: status, body: gin.H{"error": message}}
}

// resolveSession 优先使用上游已注入的非空用户，否则读取访问 Cookie 并验证 JWT。
// 错误类型或空指针上下文为服务器错误，缺少 Cookie 或令牌校验失败为未认证。
// 只有读取到令牌后才检查认证服务是否存在，成功用户写回上下文。
func resolveSession(c *gin.Context, authService *auth.AuthService, authCookies *AuthCookies) sessionOutcome {
	if userValue, exists := c.Get("user"); exists {
		user, ok := userValue.(*auth.User)
		if !ok || user == nil {
			return denied(http.StatusInternalServerError, "Invalid user context")
		}
		return sessionOutcome{user: user}
	}

	token, err := authCookies.ReadAccess(c)
	if err != nil || token == "" {
		return denied(http.StatusUnauthorized, "Missing or invalid access token")
	}

	if authService == nil {
		return denied(http.StatusInternalServerError, "Authentication service unavailable")
	}

	user, err := authService.ValidateToken(token)
	if err != nil {
		return denied(http.StatusUnauthorized, "Invalid or expired token")
	}

	c.Set("user", user)
	return sessionOutcome{user: user}
}

// RequireAuth 先解析身份并检查强制改密限制，再对无用户结果写入拒绝响应。
// 身份存在才执行后续处理，不在此限制具体账号角色。
func RequireAuth(authService *auth.AuthService, authCookies *AuthCookies) gin.HandlerFunc {
	return func(c *gin.Context) {
		r := resolveSession(c, authService, authCookies)
		if rejectPasswordChange(c, r.user) {
			return
		}
		if r.user == nil {
			c.AbortWithStatusJSON(r.status, r.body)
			return
		}
		c.Next()
	}
}

// RequireAdmin 要求上下文存在管理员角色用户，本入口不自行解析 Cookie。
// 不存在、类型错误或角色不符均返回禁止访问；调用链应提供非空用户指针。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userValue, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": errUserNotAuthenticated})
			return
		}

		user, ok := userValue.(*auth.User)
		if !ok || !user.HasRole(auth.RoleAdmin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
			return
		}

		c.Next()
	}
}

// RequireCookieSession 拒绝带 API 密钥认证标记的请求，供凭据管理等路由排除密钥调用。
// 它不直接验证 Cookie，代理注入或其他未带此标记的身份仍能通过该项检查，
// 所以调用方需与严格认证中间件组合使用。
func RequireCookieSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsAPIKeyAuth(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "this endpoint requires a browser session, not an API key"})
			return
		}
		c.Next()
	}
}

// OptionalAuth 对已存在的上下文用户直接放行，不在此核对其类型。
// 没有 Cookie 或 Cookie 校验失败仍继续，只有读到令牌却没有认证服务时返回服务器错误。
// 本入口不执行强制改密门禁，成功解析的用户写入上下文。
func OptionalAuth(authService *auth.AuthService, authCookies *AuthCookies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get("user"); exists {
			// 可选认证不消费已有值；角色门禁或 MustGetUser 负责其后的具体检查。
			c.Next()
			return
		}
		token, err := authCookies.ReadAccess(c)
		if err != nil || token == "" {
			c.Next()
			return
		}
		if authService == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Authentication service unavailable"})
			return
		}
		if user, err := authService.ValidateToken(token); err == nil {
			c.Set("user", user)
		}
		c.Next()
	}
}

// RequireEditorOrAdmin 读取已有用户并要求编辑者或管理员角色，其他情况返回禁止访问。
// 它不验证令牌有效期，调用链应先完成身份解析并提供非空用户指针。
func RequireEditorOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userValue, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": errUserNotAuthenticated})
			return
		}

		user, ok := userValue.(*auth.User)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": errUserNotAuthenticated})
			return
		}

		if user.HasRole(auth.RoleAdmin) || user.HasRole(auth.RoleEditor) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Editor or Admin role required"})
	}
}
