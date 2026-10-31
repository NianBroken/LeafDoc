// 本文件把共享路由基础设施交给各业务模块，统一认证组与公开读取组的中间件顺序。
package http

import (
	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	authmiddleware "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/gin-gonic/gin"
)

// RouterContext 提供注册路由所需的分组、Cookie 管理器和全局选项。
// 各业务注册器决定具体路径与额外权限检查，共用组函数不自动赋予编辑或管理员权限。
type RouterContext struct {
	// 根 Gin 引擎，带基础路径的业务路由通过 Base 注册。
	Engine *gin.Engine
	// 已应用 BasePath 的路由接口，具体业务路径在其后追加。
	Base gin.IRouter
	// 负责访问与刷新 JWT Cookie 的读写。
	AuthCookies *authmiddleware.AuthCookies
	// 负责 CSRF Cookie 的签发与读取，中间件另行比较请求令牌。
	CSRFCookie *security.CSRFCookie
	// 保存路由装配时传入的全局选项及运行时服务入口。
	Opts RouterOptions
}

// APIAuthGroup 在 /api 下先执行严格身份认证，再执行 CSRF 检查。
// 具体角色限制由业务追加；CSRF 中间件对 GET、HEAD、OPTIONS 和已解析 API 密钥请求放行。
func (ctx RouterContext) APIAuthGroup(authService *coreauth.AuthService) *gin.RouterGroup {
	g := ctx.Base.Group("/api")
	g.Use(
		authmiddleware.RequireAuth(authService, ctx.AuthCookies),
		security.CSRFMiddleware(ctx.CSRFCookie),
	)
	return g
}

// APIReadGroup 在 /api 下使用动态公开读取开关，已认证身份与密码状态限制仍保留。
// 此函数本身不限制 HTTP 方法，业务注册器负责只把允许公开的读取路由接入此组。
func (ctx RouterContext) APIReadGroup(authService *coreauth.AuthService) *gin.RouterGroup {
	g := ctx.Base.Group("/api")
	g.Use(
		authmiddleware.RequireAuthOrPublicRead(authService, ctx.AuthCookies, ctx.Opts.PublicAccess),
		security.CSRFMiddleware(ctx.CSRFCookie),
	)
	return g
}

// RouteRegistrar 是各业务模块的路由装配接口，接收同一共享上下文。
type RouteRegistrar interface {
	RegisterRoutes(ctx RouterContext)
}
