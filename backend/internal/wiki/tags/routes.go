package tags

import (
	"net/http"
	"strconv"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/gin-gonic/gin"
)

// Routes 将标签候选与按标签查页两个读取用例暴露为 HTTP 接口。
type Routes struct {
	getTags        *GetTagsUseCase
	getPagesByTags *GetPagesByTagsUseCase
	authService    *coreauth.AuthService
}

// RoutesConfig 提供标签用例与读取鉴权服务，不保存单次请求的筛选参数。
type RoutesConfig struct {
	GetTags        *GetTagsUseCase
	GetPagesByTags *GetPagesByTagsUseCase
	AuthService    *coreauth.AuthService
}

// NewRoutes 保存依赖，实际路由由注册入口统一挂载。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		getTags:        cfg.GetTags,
		getPagesByTags: cfg.GetPagesByTags,
		authService:    cfg.AuthService,
	}
}

// RegisterRoutes 为两个标签 GET 接口共用动态公开读取权限。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	// 公开模式按请求读取，注册时不将接口固定为公开或私有。
	readGroup := ctx.APIReadGroup(r.authService)
	readGroup.GET("/tags", r.handleGetTags)
	readGroup.GET("/tags/pages", r.handleGetPagesByTags)
}

// handleGetTags 解析筛选、已选标签和整数条数，条数范围由用例归一化。
// 仅整数转换失败在此返回 400，成功后将计数列表直接编码为响应。
func (r *Routes) handleGetTags(c *gin.Context) {
	filter := c.DefaultQuery("q", "")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		respondWithTagsBadRequest(c, ErrCodeTagsInvalidLimit, "Invalid limit value", "invalid limit value")
		return
	}

	out, err := r.getTags.Execute(c.Request.Context(), GetTagsInput{
		Filter:   filter,
		Selected: queryTags(c, "selected"),
		Limit:    limit,
	})
	if err != nil {
		respondWithTagsError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Tags)
}

// handleGetPagesByTags 合并重复 tags 参数与逗号列表，没有有效条目时返回 400。
// 页面查询和标签大小写规范化由用例执行。
func (r *Routes) handleGetPagesByTags(c *gin.Context) {
	tagList := queryTags(c, "tags")
	if len(tagList) == 0 {
		respondWithTagsBadRequest(c, ErrCodeTagsMissingParam, "Query parameter 'tags' is required", "query parameter tags is required")
		return
	}

	out, err := r.getPagesByTags.Execute(c.Request.Context(), GetPagesByTagsInput{Tags: tagList})
	if err != nil {
		respondWithTagsError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Pages)
}

// splitTags 拆分单个查询值，去除两端空白并跳过空项，保留大小写及重复条目。
func splitTags(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

// queryTags 顺序展开同名参数的所有值，不将仅第一项作为完整选择。
func queryTags(c *gin.Context, key string) []string {
	values := c.QueryArray(key)
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, splitTags(value)...)
	}
	return result
}
