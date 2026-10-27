package tags

import (
	httpinternal "github.com/perber/wiki/internal/http"
	"net/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeTagsInternal     = "tags_internal_error"
	ErrCodeTagsMissingParam = "tags_missing_param"
	ErrCodeTagsInvalidLimit = "tags_invalid_limit"
)

type tagsErrorResponse = httpinternal.ErrorResponse

type tagsErrorDetail = httpinternal.ErrorDetail

func respondWithTagsError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		httpinternal.RespondError(c, tagsErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	httpinternal.RespondError(c, http.StatusInternalServerError, ErrCodeTagsInternal, "Internal server error", "internal server error")
}

func respondWithTagsBadRequest(c *gin.Context, code, message, template string) {
	httpinternal.RespondError(c, http.StatusBadRequest, code, message, template)
}

func tagsErrorStatus(code string) int {
	switch code {
	case ErrCodeTagsMissingParam, ErrCodeTagsInvalidLimit:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
