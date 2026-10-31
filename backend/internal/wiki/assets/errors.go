package assets

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// 资源错误代码区分请求输入、页面或文件缺失、名称冲突及未分类内部失败。
// 代码参与 HTTP 状态映射和前端翻译，不依据错误消息字符串进行判断。
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
	ErrCodeAssetInternalError    = "asset_internal_error"
)

// respondWithAssetStatusError 通过 HTTP 公共入口输出结构化错误及翻译参数。
func respondWithAssetStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithAssetError 保留领域本地化错误信息，未知错误使用固定 500 内容。
func respondWithAssetError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithAssetStatusError(c, assetErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	respondWithAssetStatusError(c, http.StatusInternalServerError, ErrCodeAssetInternalError, "Asset request failed", "asset request failed")
}

// assetErrorStatus 将大小超限映射为 413，缺失映射为 404，名称占用映射为 409。
// 明确列出的输入错误返回 400，其余错误代码返回 500。
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
