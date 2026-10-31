package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/gin-gonic/gin"
)

// RemoteUserConfig 指定受信代理提交身份的请求头、账号创建规则和用户服务入口。
// 身份请求头只有在功能开启且实际连接对端属于信任集合时才参与鉴权。
type RemoteUserConfig struct {
	Enabled bool
	// HeaderName 提供用户名或邮箱，读取后去除两端空白，空值交给后续鉴权。
	HeaderName string
	// AutoCreate 控制未知身份是否交给用户服务创建账号，与 Enabled 分别控制。
	AutoCreate bool
	// EmailHeaderName 只在自动创建路径中读取，作为用户服务的邮箱参数。
	// 邮箱缺失时的占位地址及冲突规则由 GetOrCreateRemoteUser 统一处理。
	EmailHeaderName string
	// DefaultRole 传给自动创建入口，普通账号查找路径不使用该值。
	DefaultRole    string
	TrustedProxies *TrustedProxies
	// UserService 在每次需要解析身份时调用，使请求使用恢复流程替换后的用户存储。
	// 中间件检查函数是否为空，返回的服务由配置方保证可用。
	UserService func() *coreauth.UserService
}

// InjectRemoteUser 将受信代理声明的身份解析为用户，并写入 Gin 上下文的 user。
// 功能关闭、来源不受信或身份头为空时继续处理链，后续路由自行决定是否要求登录。
// 功能开启但缺少信任配置或服务入口时返回 500，不继续尝试其他凭据。
// 自动创建路径中的邮箱冲突返回 401，其他创建错误返回 500；普通查找错误返回 401。
// 成功时写入用户但不设置 API 密钥标记，因此不触发基于该标记的 CSRF 豁免。
func InjectRemoteUser(cfg RemoteUserConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		if cfg.TrustedProxies == nil || cfg.UserService == nil {
			slog.Default().Error("reverse proxy auth: misconfigured, TrustedProxies or UserService is nil")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Reverse proxy authentication misconfigured"})
			return
		}

		if !cfg.TrustedProxies.IsTrusted(c.Request.RemoteAddr) {
			slog.Default().Debug("reverse proxy auth: request from untrusted IP, skipping", "remote_addr", c.Request.RemoteAddr)
			c.Next()
			return
		}

		identifier := strings.TrimSpace(c.GetHeader(cfg.HeaderName))
		if identifier == "" {
			slog.Default().Debug("reverse proxy auth: trusted proxy sent no user header, skipping", "remote_addr", c.Request.RemoteAddr, "header", cfg.HeaderName)
			// 受信来源不等于已经登录。空身份不阻断公开页面，由路由权限继续检查。
			c.Next()
			return
		}

		var user *coreauth.User
		var err error
		// 两种路径都动态取得用户服务，自动创建路径另外传递邮箱与默认角色。
		if cfg.AutoCreate {
			email := strings.TrimSpace(c.GetHeader(cfg.EmailHeaderName))
			user, err = cfg.UserService().GetOrCreateRemoteUser(identifier, email, cfg.DefaultRole)
			if err != nil {
				if errors.Is(err, coreauth.ErrRemoteUserEmailConflict) {
					slog.Default().Warn("reverse proxy auth: auto-create blocked, asserted email belongs to a different user", "identifier", identifier, "remote_addr", c.Request.RemoteAddr)
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "reverse proxy auth: user not found"})
					return
				}
				slog.Default().Error("reverse proxy auth: auto-create failed", "identifier", identifier, "remote_addr", c.Request.RemoteAddr, "error", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "reverse proxy auth: failed to provision user"})
				return
			}
		} else {
			user, err = cfg.UserService().GetUserByIdentifier(identifier)
			if err != nil {
				slog.Default().Warn("reverse proxy auth: user not found", "identifier", identifier, "remote_addr", c.Request.RemoteAddr)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "reverse proxy auth: user not found"})
				return
			}
		}

		// 此处直接写入成功结果，不保留同名的上游上下文值。
		c.Set("user", user)
		c.Next()
	}
}
