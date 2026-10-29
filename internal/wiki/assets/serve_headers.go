package assets

import (
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/gin-gonic/gin"
)

// assetServeHeaders hardens the static /assets responses; see
// security.SetUntrustedFileHeaders for the rules.
func assetServeHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		security.SetUntrustedFileHeaders(c.Writer.Header(), c.Request.URL.Path)
		c.Next()
	}
}
