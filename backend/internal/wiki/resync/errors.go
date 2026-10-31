// 本文件将重建请求的业务错误转换为统一的 HTTP 错误响应。
package wikiresync

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

const (
	ErrCodeResyncAlreadyRunning = "resync_already_running"
	ErrCodeResyncInternalError  = "resync_internal_error"
)

// ErrResyncAlreadyRunning 表示任务运行占用尚未释放，接口以冲突状态拒绝重复请求。
var ErrResyncAlreadyRunning = sharederrors.NewLocalizedError(
	ErrCodeResyncAlreadyRunning,
	"A sync is already in progress",
	"a sync is already in progress",
	nil,
)

// ResyncErrorResponse 复用公共错误结构，保持错误码和翻译参数的 JSON 形状一致。
type ResyncErrorResponse = httpinternal.ErrorResponse

// respondWithResyncStatusError 将状态码、原始文字与翻译参数交给公共响应器。
func respondWithResyncStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithResyncError 保留可翻译错误的全部参数，其他错误统一返回内部失败文字，
// 不把底层错误内容直接写入响应体。
func respondWithResyncError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithResyncStatusError(c, resyncErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}
	respondWithResyncStatusError(c, http.StatusInternalServerError, ErrCodeResyncInternalError, "Resync request failed", "resync request failed")
}

// resyncErrorStatus 只将重复运行映射为 409，其余错误码均映射为 500。
func resyncErrorStatus(code string) int {
	switch code {
	case ErrCodeResyncAlreadyRunning:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
