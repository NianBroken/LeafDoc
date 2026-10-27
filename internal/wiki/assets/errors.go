package assets

import (
	httpinternal "github.com/perber/wiki/internal/http"
	"net/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeAssetFileTooLarge     = "asset_file_too_large"
	ErrCodeAssetMissingFile      = "asset_missing_file"
	ErrCodeAssetMissingName      = "asset_missing_name"
	ErrCodeAssetPageNotFound     = "asset_page_not_found"
	ErrCodeAssetNotFound         = "asset_not_found"
	ErrCodeAssetAlreadyExists    = "asset_already_exists"
	ErrCodeAssetInvalidExtension = "asset_invalid_extension"
	ErrCodeAssetInvalidName      = "asset_invalid_name"
	ErrCodeAssetInvalidPayload   = "asset_invalid_payload"
	ErrCodeAssetUploadFailed     = "asset_upload_failed"
	ErrCodeAssetDeleteFailed     = "asset_delete_failed"
	ErrCodeAssetRenameFailed     = "asset_rename_failed"
	ErrCodeAssetInternalError    = "asset_internal_error"
)

// AssetErrorResponse is the structured JSON error body returned by asset endpoints.
type AssetErrorResponse = httpinternal.ErrorResponse

// AssetErrorDetail carries the localization-ready error data.
type AssetErrorDetail = httpinternal.ErrorDetail

func respondWithAssetStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithAssetError maps errors to JSON responses for asset endpoints.
func respondWithAssetError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithAssetStatusError(c, assetErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	respondWithAssetStatusError(c, http.StatusInternalServerError, ErrCodeAssetInternalError, "Asset request failed", "asset request failed")
}

func assetErrorStatus(code string) int {
	switch code {
	case ErrCodeAssetFileTooLarge:
		return http.StatusRequestEntityTooLarge
	case ErrCodeAssetPageNotFound, ErrCodeAssetNotFound:
		return http.StatusNotFound
	case ErrCodeAssetAlreadyExists:
		return http.StatusConflict
	case ErrCodeAssetMissingFile, ErrCodeAssetMissingName, ErrCodeAssetInvalidPayload, ErrCodeAssetInvalidExtension, ErrCodeAssetInvalidName:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
