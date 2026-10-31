// 本文件维护恢复接口的错误码和响应转换，保留来自快照校验的业务错误。
package wikirestore

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

const (
	ErrCodeRestoreNotEnabled      = "restore_not_enabled"
	ErrCodeRestoreNotIntervenable = "restore_not_needs_intervention"
	ErrCodeRestoreInternalError   = "restore_internal_error"
	ErrCodeRestoreUploadTooLarge  = "restore_upload_too_large"
	ErrCodeRestoreUploadInvalid   = "restore_upload_invalid"
	ErrCodeRestoreMissingFile     = "restore_missing_file"
	ErrCodeRestoreFileOpenFailed  = "restore_file_open_failed"
)

// RestoreErrorResponse 复用公共 JSON 错误结构，保持错误与翻译字段一致。
type RestoreErrorResponse = httpinternal.ErrorResponse

// respondWithRestoreStatusError 按指定状态输出业务文字、翻译模板与参数。
func respondWithRestoreStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithRestoreError 保留可翻译错误的参数，普通错误使用固定内部失败响应。
func respondWithRestoreError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithRestoreStatusError(c, restoreErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}
	respondWithRestoreStatusError(c, http.StatusInternalServerError, ErrCodeRestoreInternalError, "Restore request failed", "restore request failed")
}

// restoreErrorStatus 将任务占用与介入状态冲突映射为 409，
// 快照缺失与无效标识分别映射为 404 和 400，其他业务错误使用 500。
func restoreErrorStatus(code string) int {
	switch code {
	case "restore_already_running", "restore_needs_intervention":
		return http.StatusConflict
	case ErrCodeRestoreNotIntervenable:
		return http.StatusConflict
	// 保留快照路径校验使用的错误码，与快照接口的状态映射一致。
	case "snapshot_not_found":
		return http.StatusNotFound
	case "snapshot_invalid_id":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
