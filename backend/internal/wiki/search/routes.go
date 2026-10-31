package search

import (
	"net/http"
	"strconv"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/gin-gonic/gin"
)

// Routes 提供搜索结果与索引状态两个读取入口，共享当前鉴权服务。
type Routes struct {
	search            *SearchUseCase
	getIndexingStatus *GetIndexingStatusUseCase
	authService       *coreauth.AuthService
}

// RoutesConfig 将查询用例与状态快照用例交给路由层，不在这里执行索引。
type RoutesConfig struct {
	Search            *SearchUseCase
	GetIndexingStatus *GetIndexingStatusUseCase
	AuthService       *coreauth.AuthService
}

// NewRoutes 保存搜索路由依赖，具体 URL 在 RegisterRoutes 中挂载。
func NewRoutes(cfg RoutesConfig) *Routes {
	return &Routes{
		search:            cfg.Search,
		getIndexingStatus: cfg.GetIndexingStatus,
		authService:       cfg.AuthService,
	}
}

// RegisterRoutes 将两个 GET 接口挂到动态公开读取组，权限随每次请求读取当前设置。
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	// 公开访问状态不在注册时固定，因此切换设置无需重建路由。
	readGroup := ctx.APIReadGroup(r.authService)
	readGroup.GET("/search/status", r.handleGetIndexingStatus)
	readGroup.GET("/search", r.handleSearch)
}

// handleSearch 接受关键词或非空标签列表，缺少两者时返回 400。
// 分页参数只检查是否可转换为整数，负值或零值继续交给用例与索引规则处理。
func (r *Routes) handleSearch(c *gin.Context) {
	query := c.Query("q")
	tags := queryTags(c, "tags")
	if query == "" && len(tags) == 0 {
		respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchMissingQuery, "Query parameter 'q' is required", "query parameter q is required")
		return
	}

	// 关键词不在这里去除空白，标签解析会拆分逗号并丢弃全空白项。
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchInvalidOffset, "Invalid offset value", "invalid offset value")
		return
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchInvalidLimit, "Invalid limit value", "invalid limit value")
		return
	}

	out, err := r.search.Execute(c.Request.Context(), SearchInput{Query: query, Tags: tags, Offset: offset, Limit: limit})
	if err != nil {
		respondWithSearchError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Result)
}

// handleGetIndexingStatus 返回共享状态的快照，状态为空时将空值编码到响应。
func (r *Routes) handleGetIndexingStatus(c *gin.Context) {
	out := r.getIndexingStatus.Execute(c.Request.Context())
	c.JSON(http.StatusOK, out.Status)
}

// splitTags 拆分单个查询值中的逗号标签并去除边缘空白，不去重也不转换大小写。
func splitTags(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

// queryTags 按参数出现顺序合并重复查询参数，每个参数值还可包含多个逗号分隔标签。
// 完全没有该参数时返回 nil，存在但只含空项时返回空切片。
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
