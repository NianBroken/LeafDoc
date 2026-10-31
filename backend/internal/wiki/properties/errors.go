package properties

import (
	"net/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/gin-gonic/gin"
)

// 属性错误代码区分键缺失、值缺失、条数格式错误与内部失败，供响应映射使用。
const (
	ErrCodePropertiesInternal     = "properties_internal_error"
	ErrCodePropertiesMissingKey   = "properties_missing_key"
	ErrCodePropertiesMissingValue = "properties_missing_value"
	ErrCodePropertiesInvalidLimit = "properties_invalid_limit"
)

// respondWithPropertiesError 保留已知本地化错误，未分类错误不向客户端输出底层文本。
func respondWithPropertiesError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		httpinternal.RespondError(c, propertiesErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	httpinternal.RespondError(c, http.StatusInternalServerError, ErrCodePropertiesInternal, "Internal server error", "internal server error")
}

// respondWithPropertiesBadRequest 将路由输入错误作为固定 400 结构化响应输出。
func respondWithPropertiesBadRequest(c *gin.Context, code, message, template string) {
	httpinternal.RespondError(c, http.StatusBadRequest, code, message, template)
}

// propertiesErrorStatus 仅将列出的输入问题归为 400，其他领域代码保留内部失败状态。
func propertiesErrorStatus(code string) int {
	switch code {
	case ErrCodePropertiesMissingKey, ErrCodePropertiesMissingValue, ErrCodePropertiesInvalidLimit:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
