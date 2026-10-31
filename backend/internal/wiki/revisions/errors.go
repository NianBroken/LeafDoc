package revisions

import (
	"errors"
	"net/http"
	"os"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// 历史错误代码区分输入、版本及页面缺失、资源读取和服务状态，供各处理器构造响应。
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

// respondWithRevisionStatusError 保留调用方明确指定的状态和本地化字段。
func respondWithRevisionStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithRevisionError 优先映射本地化代码，普通文件不存在错误返回通用历史资源 404。
// 未分类错误使用固定 500 文本，不直接输出底层原因。
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

// revisionErrorStatus 只识别这里明确列出的恢复及预览代码，其余本地化代码返回 500。
// 处理器直接指定状态的载荷错误不经过此表。
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
