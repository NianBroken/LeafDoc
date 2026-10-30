package instancesettings

import (
	"net/http"

	"github.com/NianBroken/LeafDoc/internal/core/publicaccess"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/gin-gonic/gin"
)

const (
	// ErrCodeInvalidPayload is returned for a malformed request body.
	ErrCodeInvalidPayload = "instance_settings_invalid_payload"
	// ErrCodeInternal is the fallback for an unclassified failure.
	ErrCodeInternal = "instance_settings_internal_error"
)

func respondWithStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithError maps a domain error to a structured JSON response.
func respondWithError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithStatusError(c, statusForCode(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}
	respondWithStatusError(c, http.StatusInternalServerError, ErrCodeInternal,
		"Instance settings request failed", "instance settings request failed")
}

func statusForCode(code string) int {
	switch code {
	case publicaccess.ErrCodeEnvManaged:
		return http.StatusConflict
	case ErrCodeInvalidPayload, "toc_display_invalid_mode":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
