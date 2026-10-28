package revisions

import (
	"errors"
	"net/http"
	"os"

	httpinternal "github.com/perber/wiki/internal/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeRevisionNotFound                    = "revision_not_found"
	ErrCodeRevisionInvalidPageID               = "revision_invalid_page_id"
	ErrCodeRevisionInvalidRevisionID           = "revision_invalid_revision_id"
	ErrCodeRevisionInvalidLimit                = "revision_invalid_limit"
	ErrCodeRevisionCompareInvalidRequest       = "revision_compare_invalid_request"
	ErrCodeRevisionRestoreInvalidPageID        = "revision_restore_invalid_page_id"
	ErrCodeRevisionRestoreInvalidRevision      = "revision_restore_invalid_revision"
	ErrCodeRevisionRestoreRevisionNotFound     = "revision_restore_revision_not_found"
	ErrCodeRevisionRestorePageNotFound         = "revision_restore_page_not_found"
	ErrCodeRevisionServiceUnavailable          = "revision_service_unavailable"
	ErrCodeRevisionPreviewAssetNotFound        = "revision_preview_asset_not_found"
	ErrCodeRevisionPreviewAssetInvalidName     = "revision_preview_asset_invalid_name"
	ErrCodeRevisionPreviewAssetBlobUnavailable = "revision_preview_asset_blob_unavailable"
	ErrCodeRevisionInternalError               = "revision_internal_error"
)

func respondWithRevisionStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithRevisionError is the central error handler for revision endpoints.
func respondWithRevisionError(c *gin.Context, err error) {
	if localized, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithRevisionStatusError(c, revisionErrorStatus(localized.Code), localized.Code, localized.Message, localized.Template, localized.Args...)
		return
	}

	switch {
	case errors.Is(err, os.ErrNotExist):
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, "Revision resource not found", "revision resource not found")
	default:
		respondWithRevisionStatusError(c, http.StatusInternalServerError, ErrCodeRevisionInternalError, "Revision request failed", "revision request failed")
	}
}

func revisionErrorStatus(code string) int {
	switch code {
	case ErrCodeRevisionRestoreRevisionNotFound, ErrCodeRevisionRestorePageNotFound, ErrCodeRevisionPreviewAssetNotFound:
		return http.StatusNotFound
	case ErrCodeRevisionRestoreInvalidPageID, ErrCodeRevisionRestoreInvalidRevision, ErrCodeRevisionPreviewAssetInvalidName:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
