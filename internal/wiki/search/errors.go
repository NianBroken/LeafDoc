package search

import (
	"net/http"

	httpinternal "github.com/perber/wiki/internal/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeSearchUnavailable   = "search_unavailable"
	ErrCodeSearchInternal      = "search_internal_error"
	ErrCodeSearchMissingQuery  = "search_missing_query"
	ErrCodeSearchInvalidOffset = "search_invalid_offset"
	ErrCodeSearchInvalidLimit  = "search_invalid_limit"
)

func respondWithSearchStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithSearchError maps errors to JSON responses for search endpoints.
func respondWithSearchError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithSearchStatusError(c, searchErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	respondWithSearchStatusError(c, http.StatusInternalServerError, ErrCodeSearchInternal, "Failed to perform search", "failed to perform search")
}

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
