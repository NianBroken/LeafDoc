package search

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// 搜索错误代码区分服务缺失、内部失败与查询分页参数错误，供状态及翻译映射使用。
const (
	ErrCodeSearchUnavailable   = "search_unavailable"
	ErrCodeSearchInternal      = "search_internal_error"
	ErrCodeSearchMissingQuery  = "search_missing_query"
	ErrCodeSearchInvalidOffset = "search_invalid_offset"
	ErrCodeSearchInvalidLimit  = "search_invalid_limit"
)

// respondWithSearchStatusError 交由 HTTP 公共入口输出结构化搜索错误。
func respondWithSearchStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithSearchError 保留本地化错误的参数，普通内部错误不向客户端暴露底层文本。
func respondWithSearchError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithSearchStatusError(c, searchErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	respondWithSearchStatusError(c, http.StatusInternalServerError, ErrCodeSearchInternal, "Failed to perform search", "failed to perform search")
}

// searchErrorStatus 将服务不可用映射为 503，明确的输入错误映射为 400，其余为 500。
func searchErrorStatus(code string) int {
	switch code {
	case ErrCodeSearchUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeSearchMissingQuery, ErrCodeSearchInvalidOffset, ErrCodeSearchInvalidLimit:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
