package auth

import (
	"net/http"
	"time"

	"github.com/NianBroken/LeafDoc/internal/http/middleware/utils"
	"github.com/gin-gonic/gin"
)

// AuthCookies 统一读写访问令牌和刷新令牌的 Cookie。
// 两种令牌分别使用自己的有效期，安全属性由当前请求与 AllowInsecure 共同确定。
// 这里负责传递令牌文本，签名、有效期和会话撤销由鉴权服务检查。
type AuthCookies struct {
	AllowInsecure bool
	SameSite      http.SameSite
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

// NewAuthCookies 保存调用方给出的有效期，并使用 Lax 跨站发送策略。
// 构造过程不校验有效期，写入时以整秒设置 MaxAge。
func NewAuthCookies(allowInsecure bool, accessTTL, refreshTTL time.Duration) *AuthCookies {
	return &AuthCookies{
		AllowInsecure: allowInsecure,
		SameSite:      http.SameSiteLaxMode,
		AccessTTL:     accessTTL,
		RefreshTTL:    refreshTTL,
	}
}

// cookieNames 为安全请求选择带 __Host- 前缀的名称，普通请求使用另一组名称。
// 读写均只针对所选的一组，不在两组之间搜索或回退。
func (a *AuthCookies) cookieNames(secure bool) (access, refresh string) {
	if secure {
		return "__Host-leafdoc_at", "__Host-leafdoc_rt"
	}
	return "leafdoc_at", "leafdoc_rt"
}

// Set 在安全检查通过后向响应写入两个独立的 Set-Cookie。
// 路径固定为根路径，不设置 Domain，HttpOnly 阻止页面脚本直接读取令牌。
// 有效期转换为整数秒，未达到整秒的部分不写入 MaxAge。
func (a *AuthCookies) Set(c *gin.Context, accessToken, refreshToken string) error {
	secure, err := utils.RequireSecure(c, a.AllowInsecure)
	if err != nil {
		return err
	}

	accessName, refreshName := a.cookieNames(secure)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     accessName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: a.SameSite,
		MaxAge:   int(a.AccessTTL.Seconds()),
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshName,
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: a.SameSite,
		MaxAge:   int(a.RefreshTTL.Seconds()),
	})

	return nil
}

// Clear 通过负的 MaxAge 要求浏览器删除当前安全模式对应的两个 Cookie。
// 该操作只改变响应，不撤销数据库中的会话，也不清除另一种模式下的名称。
func (a *AuthCookies) Clear(c *gin.Context) error {
	secure, err := utils.RequireSecure(c, a.AllowInsecure)
	if err != nil {
		return err
	}

	accessName, refreshName := a.cookieNames(secure)

	// 删除时保持名称、根路径及安全属性与写入路径一致。
	expire := func(name string) {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: a.SameSite,
			MaxAge:   -1,
		})
	}

	expire(accessName)
	expire(refreshName)
	return nil
}

// ReadAccess 读取当前请求对应的访问令牌 Cookie，原样返回文本及读取错误。
// 不满足安全要求时直接返回错误，不继续读取 Cookie。
func (a *AuthCookies) ReadAccess(c *gin.Context) (string, error) {
	secure, err := utils.RequireSecure(c, a.AllowInsecure)
	if err != nil {
		return "", err
	}
	accessName, _ := a.cookieNames(secure)
	return c.Cookie(accessName)
}

// ReadRefresh 以同样的安全规则读取刷新令牌，令牌内容交给会话服务验证。
func (a *AuthCookies) ReadRefresh(c *gin.Context) (string, error) {
	secure, err := utils.RequireSecure(c, a.AllowInsecure)
	if err != nil {
		return "", err
	}
	_, refreshName := a.cookieNames(secure)
	return c.Cookie(refreshName)
}
