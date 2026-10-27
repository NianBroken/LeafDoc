package wikiresync

import (
	httpinternal "github.com/perber/wiki/internal/http"
	"net/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeResyncAlreadyRunning = "resync_already_running"
	ErrCodeResyncInternalError  = "resync_internal_error"
)

var ErrResyncAlreadyRunning = sharederrors.NewLocalizedError(
	ErrCodeResyncAlreadyRunning,
	"A sync is already in progress",
	"a sync is already in progress",
	nil,
)

// ResyncErrorResponse is the structured JSON error body returned by resync endpoints.
type ResyncErrorResponse = httpinternal.ErrorResponse

// ResyncErrorDetail carries the localization-ready error data.
type ResyncErrorDetail = httpinternal.ErrorDetail

func respondWithResyncStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

func respondWithResyncError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithResyncStatusError(c, resyncErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}
	respondWithResyncStatusError(c, http.StatusInternalServerError, ErrCodeResyncInternalError, "Resync request failed", "resync request failed")
}

func resyncErrorStatus(code string) int {
	switch code {
	case ErrCodeResyncAlreadyRunning:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
