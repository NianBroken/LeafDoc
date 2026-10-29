package auth

import (
	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/gin-gonic/gin"
)

func InjectPublicEditor(authDisabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authDisabled {
			if _, exists := c.Get("user"); !exists {
				c.Set("user", &auth.User{
					ID:       "public-editor",
					Username: "public-editor",
					Role:     auth.RoleEditor,
				})
			}
		}
		c.Next()
	}
}
