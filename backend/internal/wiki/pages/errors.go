package pages

import (
	"errors"
	"fmt"
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/gin-gonic/gin"
)

// 页面接口使用的稳定错误码，状态映射和客户端本地化按这些值识别失败原因。
const (
	ErrCodePageNotFound          = "page_not_found"
	ErrCodePageParentNotFound    = "page_parent_not_found"
	ErrCodePageSlugConflict      = "page_slug_conflict"
	ErrCodePageHasChildren       = "page_has_children"
	ErrCodePageCircularMove      = "page_circular_move"
	ErrCodePageCannotMoveToSelf  = "page_cannot_move_to_self"
	ErrCodePageRootOperation     = "page_root_operation"
	ErrCodePageConvertNotAllowed = "page_convert_not_allowed"
	ErrCodePageInternalError     = "page_internal_error"
	ErrCodePageMissingPath       = "page_missing_path"
	ErrCodePageMissingID         = "page_missing_id"
	ErrCodePageMissingTitle      = "page_missing_title"
	ErrCodePageVersionRequired   = "page_version_required"
	ErrCodePageVersionConflict   = "page_version_conflict"
	ErrCodePageInvalidRequest    = "page_invalid_request"
	ErrCodePageInvalidPayload    = "page_invalid_payload"
	ErrCodePageInvalidTargetKind = "page_invalid_target_kind"
)

// newPageRootOperationError 构造禁止根节点操作的结构化错误，操作名称作为模板参数。
func newPageRootOperationError(operation string) *sharederrors.LocalizedError {
	return sharederrors.NewLocalizedError(
		ErrCodePageRootOperation,
		fmt.Sprintf("cannot %s root page", operation),
		"cannot %s root page",
		nil, operation,
	)
}

// respondWithPageStatusError 将状态、代码和翻译模板统一交给 HTTP 错误响应入口。
func respondWithPageStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithPageError 先处理带本地化字段的错误，再处理字段校验错误。
// 其余错误按 errors.Is 匹配树服务分类，未知错误使用内部错误码和原错误文字。
// 版本冲突返回 409，缺失版本和无效操作返回 400，不在此重试请求。
func respondWithPageError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithPageStatusError(c, pageErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	var vErr *sharederrors.ValidationErrors
	if errors.As(err, &vErr) {
		httpinternal.RespondValidationErrors(c, vErr)
		return
	}

	switch {
	case errors.Is(err, tree.ErrPageNotFound):
		respondWithPageStatusError(c, http.StatusNotFound, ErrCodePageNotFound, "Page not found", "page not found")
	case errors.Is(err, tree.ErrParentNotFound):
		respondWithPageStatusError(c, http.StatusNotFound, ErrCodePageParentNotFound, "Parent page not found", "parent page not found")
	case errors.Is(err, tree.ErrPageHasChildren):
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageHasChildren, "Page has children, use recursive delete", "page has children")
	case errors.Is(err, tree.ErrPageAlreadyExists):
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageSlugConflict, "Page already exists", "page already exists")
	case errors.Is(err, tree.ErrMovePageCircularReference):
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageCircularMove, "Move would create a circular reference", "circular reference detected")
	case errors.Is(err, tree.ErrPageCannotBeMovedToItself):
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageCannotMoveToSelf, "Page cannot be moved to itself", "page cannot be moved to itself")
	case errors.Is(err, tree.ErrConvertNotAllowed):
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageConvertNotAllowed, "Convert operation not allowed", "convert not allowed")
	case errors.Is(err, tree.ErrInvalidOperation):
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, err.Error(), "invalid operation")
	case errors.Is(err, tree.ErrVersionConflict):
		respondWithPageStatusError(c, http.StatusConflict, ErrCodePageVersionConflict, "Page was changed by another request", "page was changed by another request")
	case errors.Is(err, tree.ErrVersionRequired):
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageVersionRequired, "Page version is required", "page version is required")
	case errors.Is(err, tree.ErrTreeNotLoaded):
		respondWithPageStatusError(c, http.StatusInternalServerError, ErrCodePageInternalError, "Tree not loaded", "tree not loaded")
	default:
		respondWithPageStatusError(c, http.StatusInternalServerError, ErrCodePageInternalError, err.Error(), "internal error")
	}
}

// pageErrorStatus 将结构化错误码映射为 HTTP 状态，收藏存储不可用单独使用 503。
// 未列出的代码统一返回 500，不从错误消息推断状态。
func pageErrorStatus(code string) int {
	switch code {
	case ErrCodePageNotFound, ErrCodePageParentNotFound:
		return http.StatusNotFound
	case ErrCodePageHasChildren, ErrCodePageCircularMove, ErrCodePageCannotMoveToSelf, ErrCodePageSlugConflict,
		ErrCodePageConvertNotAllowed, ErrCodePageRootOperation, ErrCodePageVersionRequired,
		ErrCodePageMissingPath, ErrCodePageMissingID, ErrCodePageMissingTitle, ErrCodePageInvalidRequest,
		ErrCodePageInvalidPayload, ErrCodePageInvalidTargetKind:
		return http.StatusBadRequest
	case ErrCodePageVersionConflict:
		return http.StatusConflict
	case favorites.ErrCodeFavoritesStoreUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
