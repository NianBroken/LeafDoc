package links

import (
	"net/http"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

// Routes 提供单页链接状态和全站失效链接审计，二者使用不同读取权限。
type Routes struct {
	getLinkStatus  *GetLinkStatusUseCase
	getBrokenLinks *GetBrokenLinksUseCase
	authService    *coreauth.AuthService
}

// RoutesConfig 提供两个查询用例及鉴权服务，不在路由构造时重新索引链接。
type RoutesConfig struct {
	GetLinkStatus  *GetLinkStatusUseCase
	GetBrokenLinks *GetBrokenLinksUseCase
	AuthService    *coreauth.AuthService
}

// NewRoutes 保存用例依赖，具体权限在注册阶段装配。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		getLinkStatus:  cfg.GetLinkStatus,
		getBrokenLinks: cfg.GetBrokenLinks,
		authService:    cfg.AuthService,
	}
}

// RegisterRoutes 为页面状态使用公开读取组，为全站审计额外要求管理员登录。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	// 页面链接信息随实例公开访问设置切换，无需重新注册路由。
	readGroup := ctx.APIReadGroup(r.authService)
	readGroup.GET("/pages/:id/links", r.handleGetLinkStatus)

	// 全站审计不采用公开读取豁免，公开模式下仍执行管理员角色检查。
	authGroup := ctx.APIAuthGroup(r.authService)
	authGroup.GET("/links/broken", authmw.RequireAdmin(), r.handleGetBrokenLinks)
}

// handleGetLinkStatus 从路由取页面标识，成功响应直接编码链接服务状态结果。
func (r *Routes) handleGetLinkStatus(c *gin.Context) {
	pageID := c.Param("id")
	out, err := r.getLinkStatus.Execute(c.Request.Context(), GetLinkStatusInput{PageID: pageID})
	if err != nil {
		respondWithLinkError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Status)
}

// handleGetBrokenLinks 返回带 links 字段的用例结果，错误统一通过链接领域响应映射。
func (r *Routes) handleGetBrokenLinks(c *gin.Context) {
	out, err := r.getBrokenLinks.Execute(c.Request.Context())
	if err != nil {
		respondWithLinkError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}
