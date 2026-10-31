// 本文件把附件静态响应接入公共不可信文件策略，不在附件模块重复定义后缀规则。
package assets

import (
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/gin-gonic/gin"
)

// assetServeHeaders 在静态文件处理之前按请求路径设置不可信文件响应头。
// 具体扩展名、下载和沙箱规则统一由安全模块执行，避免静态资源另设一套规则。
func assetServeHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		security.SetUntrustedFileHeaders(c.Writer.Header(), c.Request.URL.Path)
		c.Next()
	}
}
