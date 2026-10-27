package branding

import (
	"errors"
	httpinternal "github.com/perber/wiki/internal/http"
	"net/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeBrandingConfigUnavailable   = "branding_config_unavailable"
	ErrCodeBrandingLogoInvalidType     = "branding_logo_invalid_type"
	ErrCodeBrandingLogoUploadFailed    = "branding_logo_upload_failed"
	ErrCodeBrandingLogoDeleteFailed    = "branding_logo_delete_failed"
	ErrCodeBrandingFaviconInvalidType  = "branding_favicon_invalid_type"
	ErrCodeBrandingFaviconUploadFailed = "branding_favicon_upload_failed"
	ErrCodeBrandingFaviconDeleteFailed = "branding_favicon_delete_failed"
	ErrCodeBrandingUpdateFailed        = "branding_update_failed"
	ErrCodeBrandingInternalError       = "branding_internal_error"
	ErrCodeBrandingInvalidPayload      = "branding_invalid_payload"
	ErrCodeBrandingLogoTooLarge        = "branding_logo_too_large"
	ErrCodeBrandingLogoMissing         = "branding_logo_missing"
	ErrCodeBrandingFaviconTooLarge     = "branding_favicon_too_large"
	ErrCodeBrandingFaviconMissing      = "branding_favicon_missing"
)

// BrandingErrorResponse is the structured JSON error body returned by branding endpoints.
type BrandingErrorResponse = httpinternal.ErrorResponse

// BrandingErrorDetail carries the localization-ready error data.
type BrandingErrorDetail = httpinternal.ErrorDetail

func respondWithBrandingStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithBrandingError maps errors to JSON responses for branding endpoints.
func respondWithBrandingError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithBrandingStatusError(c, brandingErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	var vErr *sharederrors.ValidationErrors
	if errors.As(err, &vErr) {
		httpinternal.RespondValidationErrors(c, vErr)
		return
	}

	respondWithBrandingStatusError(c, http.StatusInternalServerError, ErrCodeBrandingInternalError, "Branding request failed", "branding request failed")
}

func brandingErrorStatus(code string) int {
	switch code {
	case ErrCodeBrandingLogoInvalidType, ErrCodeBrandingFaviconInvalidType,
		ErrCodeBrandingInvalidPayload, ErrCodeBrandingLogoMissing, ErrCodeBrandingFaviconMissing:
		return http.StatusBadRequest
	case ErrCodeBrandingLogoTooLarge, ErrCodeBrandingFaviconTooLarge:
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusInternalServerError
	}
}
