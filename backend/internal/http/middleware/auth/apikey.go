// 本文件在 Cookie 身份解析前识别 LeafDoc API 密钥，并写入统一的请求用户上下文。
// 密钥格式识别只选择认证路径，真正的有效期、撤销状态和有效角色由核心服务核验。
package auth

import (
	"log/slog"
	"net/http"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/gin-gonic/gin"
)

// APIKeyConfig 将密钥解析服务与客户端请求次数限制器交给鉴权中间件。
// 服务为空时整个密钥鉴权路径跳过，限制器为空时只跳过次数限制。
type APIKeyConfig struct {
	Service *coreauth.APIKeyService
	// RateLimiter 在解析 LeafDoc 密钥前按连接对端地址登记请求。
	// 成功解析后是否清空该地址的全部记录，由限制器的 resetOnSuccess 决定。
	RateLimiter *security.KeyedLimiter
}

// InjectAPIKeyUser 识别 Bearer 请求头中的 LeafDoc 密钥，并将解析结果写入 user。
// 上游已设置 user 时直接继续，不覆盖该值，也不在这里检查其具体类型。
// 缺少密钥或令牌不符合 LeafDoc 密钥形式时继续后续 Cookie 等鉴权路径。
// 符合密钥形式的请求在解析前接受次数检查，超限返回 429，解析失败返回 401。
// 存储暂停返回 503，并按成功结果通知限制器，允许配置中的成功清空规则生效。
// 成功结果同时带有 API 密钥标记，供 CSRF 中间件放行及凭据管理路由拒绝使用。
// 实际读写权限仍由解析后的用户角色和后续路由权限中间件共同约束。
func InjectAPIKeyUser(cfg APIKeyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Service == nil {
			c.Next()
			return
		}

		// 只检查键是否存在，上游身份优先，值的类型留给实际读取身份的门禁核验。
		if _, exists := c.Get("user"); exists {
			c.Next()
			return
		}

		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" || !coreauth.LooksLikeAPIKeyToken(token) {
			c.Next()
			return
		}

		// 次数限制针对网络连接对端，不使用 Bearer 内容作为计数键。
		var limiterKey string
		if cfg.RateLimiter != nil {
			limiterKey = security.ClientKey(c)
			if !cfg.RateLimiter.Allow(limiterKey) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many API key attempts, please try again later"})
				return
			}
		}

		user, err := cfg.Service.Resolve(token)
		if loc, ok := coreauth.AsStoreUnavailableErr(err); ok {
			// 暂停存储不作为无效凭据处理。启用成功清空时删除该地址的全部历史，
			// 并非只撤销当前一次登记；关闭该配置时已经登记的请求仍留在窗口内。
			if cfg.RateLimiter != nil {
				cfg.RateLimiter.NotifyResult(limiterKey, true)
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": loc.Message})
			return
		}
		if cfg.RateLimiter != nil {
			cfg.RateLimiter.NotifyResult(limiterKey, err == nil)
		}
		if err != nil {
			slog.Default().Warn("api key auth: rejected", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired API key"})
			return
		}

		// 用户对象包含服务计算出的有效角色，标记表示密钥已经通过实际解析。
		c.Set("user", user)
		c.Set(security.APIKeyAuthContextKey, true)
		c.Next()
	}
}

// bearerToken 只接受大小写完全一致的 Bearer 前缀，去除令牌两端空白。
// 不匹配时返回空字符串，使调用方继续其他鉴权路径。
func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
