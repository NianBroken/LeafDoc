package importer

import (
	"net/http"

	httpinternal "github.com/perber/wiki/internal/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeImporterNoPlan           = "importer_no_plan"
	ErrCodeImporterExecutionRunning = "importer_execution_running"
	ErrCodeImporterStateUnavailable = "importer_state_unavailable"
	ErrCodeImporterInternalError    = "importer_internal_error"
	ErrCodeImporterUploadTooLarge   = "importer_upload_too_large"
	ErrCodeImporterMissingFile      = "importer_missing_file"
	ErrCodeImporterFileOpenFailed   = "importer_file_open_failed"

	ErrCodeImporterZipEntryTooLarge     = "importer_zip_entry_too_large"
	ErrCodeImporterZipExtractedTooLarge = "importer_zip_extracted_too_large"
	ErrCodeImporterZipRatioTooHigh      = "importer_zip_ratio_too_high"
)

func respondWithImporterStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithImporterError maps errors to JSON responses for importer endpoints.
func respondWithImporterError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithImporterStatusError(c, importerErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	respondWithImporterStatusError(c, http.StatusInternalServerError, ErrCodeImporterInternalError, "Importer request failed", "importer request failed")
}

func importerErrorStatus(code string) int {
	switch code {
	case ErrCodeImporterNoPlan:
		return http.StatusNotFound
	case ErrCodeImporterExecutionRunning:
		return http.StatusConflict
	case ErrCodeImporterStateUnavailable:
		return http.StatusInternalServerError
	case ErrCodeImporterUploadTooLarge, ErrCodeImporterZipEntryTooLarge, ErrCodeImporterZipExtractedTooLarge, ErrCodeImporterZipRatioTooHigh:
		return http.StatusRequestEntityTooLarge
	case ErrCodeImporterMissingFile, ErrCodeImporterFileOpenFailed:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
