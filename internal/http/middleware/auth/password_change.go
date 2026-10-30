package auth

import (
	"net/http"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/localization"
	"github.com/gin-gonic/gin"
)

// passwordChangeAllowed is the small set of routes required to finish or end a restricted session.
func passwordChangeAllowed(path string) bool {
	for _, suffix := range []string{"/api/auth/me", "/api/auth/login", "/api/auth/refresh-token", "/api/auth/logout", "/api/users/me/password", "/api/config", "/api/branding", "/api/settings/events"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return !strings.Contains(path, "/api/") && !strings.Contains(path, "/assets/")
}
func rejectPasswordChange(c *gin.Context, user *coreauth.User) bool {
	if user != nil && user.MustChangePassword && !passwordChangeAllowed(c.Request.URL.Path) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "auth_password_change_required", "template": "password change required", "message": localization.Message("password change required")}})
		return true
	}
	return false
}

// PasswordChangeGate resolves existing credentials before public and protected handlers alike.
func PasswordChangeGate(service *coreauth.AuthService, cookies *AuthCookies) gin.HandlerFunc {
	return func(c *gin.Context) {
		result := resolveSession(c, service, cookies)
		if rejectPasswordChange(c, result.user) {
			return
		}
		c.Next()
	}
}
