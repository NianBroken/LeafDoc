package avatar

import (
	httpinternal "github.com/perber/wiki/internal/http"
	"net/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeAvatarInvalidType   = "avatar_invalid_type"
	ErrCodeAvatarDecodeFailed  = "avatar_decode_failed"
	ErrCodeAvatarUploadFailed  = "avatar_upload_failed"
	ErrCodeAvatarDeleteFailed  = "avatar_delete_failed"
	ErrCodeAvatarInternalError = "avatar_internal_error"
)

// AvatarErrorResponse is the structured JSON error body returned by
// avatar endpoints.
type AvatarErrorResponse = httpinternal.ErrorResponse

// AvatarErrorDetail carries the localization-ready error data.
type AvatarErrorDetail = httpinternal.ErrorDetail

func respondWithAvatarStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithAvatarError maps errors to JSON responses for avatar endpoints.
func respondWithAvatarError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithAvatarStatusError(c, avatarErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	respondWithAvatarStatusError(c, http.StatusInternalServerError, ErrCodeAvatarInternalError, "Avatar request failed", "avatar request failed")
}

func avatarErrorStatus(code string) int {
	switch code {
	case ErrCodeAvatarInvalidType, ErrCodeAvatarDecodeFailed:
		return http.StatusBadRequest
	case ErrCodeAvatarUploadFailed, ErrCodeAvatarDeleteFailed, ErrCodeAvatarInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
