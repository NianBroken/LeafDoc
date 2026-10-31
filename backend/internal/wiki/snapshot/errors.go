// 本文件统一快照操作的错误响应，并将冲突、缺失和无效标识映射为 HTTP 状态。
package wikisnapshot

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

const (
	ErrCodeSnapshotNotEnabled    = "snapshot_not_enabled"
	ErrCodeSnapshotInternalError = "snapshot_internal_error"
)

// respondWithSnapshotStatusError 将翻译模板与参数一起交给公共响应器。
func respondWithSnapshotStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithSnapshotError 保留可翻译错误的业务信息，普通错误统一输出内部失败，
// 不将底层错误原文暴露给客户端。
func respondWithSnapshotError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithSnapshotStatusError(c, snapshotErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}
	respondWithSnapshotStatusError(c, http.StatusInternalServerError, ErrCodeSnapshotInternalError, "Snapshot request failed", "snapshot request failed")
}

// snapshotErrorStatus 对已排队、找不到归档和无效标识分别返回 409、404 和 400，
// 其他错误码使用 500。依赖未启用的 503 由处理器直接指定。
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
