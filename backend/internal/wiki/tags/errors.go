package tags

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// 标签错误代码将参数缺失、条数格式错误与内部失败区分，供状态及翻译处理使用。
const (
	ErrCodeTagsInternal     = "tags_internal_error"
	ErrCodeTagsMissingParam = "tags_missing_param"
	ErrCodeTagsInvalidLimit = "tags_invalid_limit"
)

// respondWithTagsError 保留本地化错误字段，未知错误使用不含底层细节的固定 500 响应。
func respondWithTagsError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		httpinternal.RespondError(c, tagsErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	httpinternal.RespondError(c, http.StatusInternalServerError, ErrCodeTagsInternal, "Internal server error", "internal server error")
}

// respondWithTagsBadRequest 输出路由参数检查产生的 400 错误。
func respondWithTagsBadRequest(c *gin.Context, code, message, template string) {
	httpinternal.RespondError(c, http.StatusBadRequest, code, message, template)
}

// tagsErrorStatus 只将明确列出的两种输入错误映射为 400，其他代码返回 500。
func tagsErrorStatus(code string) int {
	switch code {
	case ErrCodeTagsMissingParam, ErrCodeTagsInvalidLimit:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
