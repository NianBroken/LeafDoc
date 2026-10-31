// 本文件为 CSRF 双重提交检查提供浏览器可读的随机令牌，不负责判断具体请求是否允许。
package security

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/NianBroken/LeafDoc/internal/http/middleware/utils"
	"github.com/gin-gonic/gin"
)

// CSRFCookie 负责浏览器双重提交令牌的生成、读取和删除。
// Cookie 允许页面脚本读取，脚本可将值放入请求头供 CSRF 中间件比较。
type CSRFCookie struct {
	// 连接策略与认证 Cookie 共用 RequireSecure，名称据该次请求的安全模式选择。
	AllowInsecure bool
	SameSite      http.SameSite
	// TTL 仅在生成并设置新 Cookie 时使用，复用已有 Cookie 不更新其到期时间。
	TTL time.Duration
}

// NewCSRFCookie 保存有效期和连接策略，默认采用 Lax 跨站发送规则。
// 构造函数不检查有效期，写入 Cookie 时转换为整数秒。
func NewCSRFCookie(allowInsecure bool, ttl time.Duration) *CSRFCookie {
	return &CSRFCookie{
		AllowInsecure: allowInsecure,
		SameSite:      http.SameSiteLaxMode,
		TTL:           ttl,
	}
}

// cookieName 按请求安全模式选择名称，安全模式使用 __Host- 前缀。
func (c *CSRFCookie) cookieName(secure bool) string {
	if secure {
		return "__Host-leafdoc_csrf"
	}
	return "leafdoc_csrf"
}

// generateToken 读取 32 字节密码学随机数，将其编码为 64 个十六进制字符。
// 随机源读取失败时返回错误，不返回不完整的令牌。
func (c *CSRFCookie) generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Issue 在连接安全检查后复用非空 Cookie，或生成令牌并写入响应。
// 两条成功路径都通过 X-CSRF-Token 响应头返回同一个值。
func (c *CSRFCookie) Issue(ctx *gin.Context) (string, error) {
	secure, err := utils.RequireSecure(ctx, c.AllowInsecure)
	if err != nil {
		return "", err
	}

	// 已有值不重新校验格式，也不重新设置 Cookie，因此不延长它的有效期。
	if existing, err := ctx.Cookie(c.cookieName(secure)); err == nil && existing != "" {
		ctx.Header("X-CSRF-Token", existing)
		return existing, nil
	}

	token, err := c.generateToken()
	if err != nil {
		return "", err
	}

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     c.cookieName(secure),
		Value:    token,
		Path:     "/",
		HttpOnly: false, // 页面脚本需要读取该值并在后续请求中显式提交。
		Secure:   secure,
		SameSite: c.SameSite,
		MaxAge:   int(c.TTL.Seconds()),
	})

	// 响应头同时提供令牌，调用方无需只依靠读取 Cookie 获取该值。
	ctx.Header("X-CSRF-Token", token)

	return token, nil
}

// Read 检查连接后读取当前模式对应的 Cookie，不检验令牌格式或比较请求头。
func (c *CSRFCookie) Read(ctx *gin.Context) (string, error) {
	secure, err := utils.RequireSecure(ctx, c.AllowInsecure)
	if err != nil {
		return "", err
	}
	return ctx.Cookie(c.cookieName(secure))
}

// Clear 发送根路径、相同名称和安全属性的过期 Cookie，要求浏览器删除该值。
// 删除仅影响当前模式下的名称，不改变认证会话。
func (c *CSRFCookie) Clear(ctx *gin.Context) error {
	secure, err := utils.RequireSecure(ctx, c.AllowInsecure)
	if err != nil {
		return err
	}

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     c.cookieName(secure),
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: c.SameSite,
		MaxAge:   -1,
	})

	return nil
}
