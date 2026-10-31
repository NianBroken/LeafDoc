package properties

import (
	"net/http"
	"strconv"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/gin-gonic/gin"
)

// Routes 提供属性键候选及按键值查询页面两个读取接口。
type Routes struct {
	getPropertyKeys    *GetPropertyKeysUseCase
	getPagesByProperty *GetPagesByPropertyUseCase
	authService        *coreauth.AuthService
}

// RoutesConfig 将查询用例与鉴权服务交给路由，不承担属性索引建立。
type RoutesConfig struct {
	GetPropertyKeys    *GetPropertyKeysUseCase
	GetPagesByProperty *GetPagesByPropertyUseCase
	AuthService        *coreauth.AuthService
}

// NewRoutes 保存现有依赖，不执行查询或注册请求处理链。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		getPropertyKeys:    cfg.GetPropertyKeys,
		getPagesByProperty: cfg.GetPagesByProperty,
		authService:        cfg.AuthService,
	}
}

// RegisterRoutes 将两个 GET 接口挂到共享公开读取组，权限由当前实例设置决定。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	// 注册一次后每个请求重新读取公开模式，不固定捕获启动时的布尔值。
	readGroup := ctx.APIReadGroup(r.authService)
	readGroup.GET("/properties", r.handleGetPropertyKeys)
	readGroup.GET("/properties/pages", r.handleGetPagesByProperty)
}

// handleGetPropertyKeys 解析过滤文字和条数，条数不能转为整数时返回 400。
// 有效整数的范围处理交给用例，成功响应直接输出键计数数组。
func (r *Routes) handleGetPropertyKeys(c *gin.Context) {
	filter := c.DefaultQuery("q", "")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		respondWithPropertiesBadRequest(c, ErrCodePropertiesInvalidLimit, "Invalid limit value", "invalid limit value")
		return
	}

	out, err := r.getPropertyKeys.Execute(c.Request.Context(), GetPropertyKeysInput{Filter: filter, Limit: limit})
	if err != nil {
		respondWithPropertiesError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Keys)
}

// handleGetPagesByProperty 原样读取 key 与 value，必填检查及索引查询由用例执行。
func (r *Routes) handleGetPagesByProperty(c *gin.Context) {
	key := c.Query("key")
	value := c.Query("value")

	out, err := r.getPagesByProperty.Execute(c.Request.Context(), GetPagesByPropertyInput{Key: key, Value: value})
	if err != nil {
		respondWithPropertiesError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Pages)
}
