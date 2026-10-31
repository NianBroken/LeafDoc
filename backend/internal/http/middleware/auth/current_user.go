// 本文件读取认证中间件已保存的用户与认证方式，不自行校验令牌或查询数据库。
package auth

import (
	"net/http"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/gin-gonic/gin"
)

// MustGetUser 读取非空用户指针，未设置返回禁止访问，类型错误或空指针返回服务器错误。
// 失败会中止 Gin 处理链并返回 nil，调用方仍须停止后续对返回值的访问。
func MustGetUser(c *gin.Context) *auth.User {
	v, exists := c.Get("user")
	if !exists {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "User not authenticated"})
		return nil
	}

	user, ok := v.(*auth.User)
	if !ok || user == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return nil
	}

	return user
}

// TryGetUser 在缺少值、类型错误或保存空用户指针时返回 nil，不写响应或中止请求。
func TryGetUser(c *gin.Context) *auth.User {
	v, exists := c.Get("user")
	if !exists {
		return nil
	}
	user, ok := v.(*auth.User)
	if !ok {
		return nil
	}
	return user
}

// IsAPIKeyAuth 读取共享上下文键并转换为布尔值，缺失或其他类型均返回 false。
// 正常链中标记由成功解析密钥的 InjectAPIKeyUser 写入，此函数不再次验证凭据。
func IsAPIKeyAuth(c *gin.Context) bool {
	v, _ := c.Get(security.APIKeyAuthContextKey)
	b, _ := v.(bool)
	return b
}
