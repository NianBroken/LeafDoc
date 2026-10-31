// 本文件在已有身份标记为必须改密时限制可访问路径，
// 保留完成改密、退出和前端加载所需的入口，不检查首次设密标记。
package auth

import (
	"net/http"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/localization"
	"github.com/gin-gonic/gin"
)

// passwordChangeAllowed 先按后缀放行列出的接口，兼容它们前面的基础路径。
// 其余地址只要不含 /api/ 和 /assets/ 也放行，不按 HTTP 方法或具体路由名判断。
func passwordChangeAllowed(path string) bool {
	for _, suffix := range []string{"/api/auth/me", "/api/auth/login", "/api/auth/refresh-token", "/api/auth/logout", "/api/users/me/password", "/api/config", "/api/branding", "/api/settings/events"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return !strings.Contains(path, "/api/") && !strings.Contains(path, "/assets/")
}

// rejectPasswordChange 只针对非空且 MustChangePassword 为真的用户，
// 路径不允许时写入本地化结构错误并中止请求，返回 true 通知上游立即结束。
func rejectPasswordChange(c *gin.Context, user *coreauth.User) bool {
	if user != nil && user.MustChangePassword && !passwordChangeAllowed(c.Request.URL.Path) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "auth_password_change_required", "template": "password change required", "message": localization.Message("password change required")}})
		return true
	}
	return false
}

// PasswordChangeGate 尝试解析现有身份后执行路径限制。身份解析失败本身不在此拒绝，
// 后续公开或严格门禁继续处理；成功身份则受到强制改密规则约束。
func PasswordChangeGate(service *coreauth.AuthService, cookies *AuthCookies) gin.HandlerFunc {
	return func(c *gin.Context) {
		result := resolveSession(c, service, cookies)
		if rejectPasswordChange(c, result.user) {
			return
		}
		c.Next()
	}
}
