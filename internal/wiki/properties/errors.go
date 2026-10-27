package properties

import (
	httpinternal "github.com/perber/wiki/internal/http"
	"net/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodePropertiesInternal     = "properties_internal_error"
	ErrCodePropertiesMissingKey   = "properties_missing_key"
	ErrCodePropertiesMissingValue = "properties_missing_value"
	ErrCodePropertiesInvalidLimit = "properties_invalid_limit"
)

type propertiesErrorResponse = httpinternal.ErrorResponse

type propertiesErrorDetail = httpinternal.ErrorDetail

func respondWithPropertiesError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		httpinternal.RespondError(c, propertiesErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	httpinternal.RespondError(c, http.StatusInternalServerError, ErrCodePropertiesInternal, "Internal server error", "internal server error")
}

func respondWithPropertiesBadRequest(c *gin.Context, code, message, template string) {
	httpinternal.RespondError(c, http.StatusBadRequest, code, message, template)
}

func propertiesErrorStatus(code string) int {
	switch code {
	case ErrCodePropertiesMissingKey, ErrCodePropertiesMissingValue, ErrCodePropertiesInvalidLimit:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
