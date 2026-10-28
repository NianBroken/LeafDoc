package wikisnapshot

import (
	"net/http"

	httpinternal "github.com/perber/wiki/internal/http"

	"github.com/gin-gonic/gin"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

const (
	ErrCodeSnapshotNotEnabled    = "snapshot_not_enabled"
	ErrCodeSnapshotInternalError = "snapshot_internal_error"
)

func respondWithSnapshotStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

func respondWithSnapshotError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithSnapshotStatusError(c, snapshotErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}
	respondWithSnapshotStatusError(c, http.StatusInternalServerError, ErrCodeSnapshotInternalError, "Snapshot request failed", "snapshot request failed")
}

func snapshotErrorStatus(code string) int {
	switch code {
	case "snapshot_already_running":
		return http.StatusConflict
	case "snapshot_not_found":
		return http.StatusNotFound
	case "snapshot_invalid_id":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
