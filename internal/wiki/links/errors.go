package links

import (
	httpinternal "github.com/perber/wiki/internal/http"
	"net/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeLinkPageNotFound  = "link_page_not_found"
	ErrCodeLinkUnavailable   = "link_service_unavailable"
	ErrCodeLinkInternalError = "link_internal_error"
)

// LinkErrorResponse is the structured JSON error body returned by link endpoints.
type LinkErrorResponse = httpinternal.ErrorResponse

// LinkErrorDetail carries the localization-ready error data.
type LinkErrorDetail = httpinternal.ErrorDetail

func respondWithLinkStatusError(c *gin.Context, status int, code, message, template string) {
	httpinternal.RespondError(c, status, code, message, template)
}

// respondWithLinkError maps errors to JSON responses for link endpoints.
func respondWithLinkError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithLinkStatusError(c, linkErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template)
		return
	}

	respondWithLinkStatusError(c, http.StatusInternalServerError, ErrCodeLinkInternalError, "Failed to load link status", "failed to load link status")
}

func linkErrorStatus(code string) int {
	switch code {
	case ErrCodeLinkPageNotFound:
		return http.StatusNotFound
	case ErrCodeLinkUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
