package importer

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// 导入错误代码区分计划状态、上传解析和解压限制，供 HTTP 状态及消息翻译使用。
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

// respondWithImporterStatusError 通过公共 HTTP 入口输出结构化错误和翻译参数。
func respondWithImporterStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithImporterError 按已知代码映射状态，未知底层错误使用固定内部失败响应。
func respondWithImporterError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithImporterStatusError(c, importerErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	respondWithImporterStatusError(c, http.StatusInternalServerError, ErrCodeImporterInternalError, "Importer request failed", "importer request failed")
}

// importerErrorStatus 将无计划映射为 404、执行冲突映射为 409、大小及解压限制映射为 413。
// 缺文件或无法打开文件返回 400，其余状态与内部错误返回 500。
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
