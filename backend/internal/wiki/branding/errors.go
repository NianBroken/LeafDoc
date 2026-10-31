package branding

import (
	"errors"
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// 品牌错误代码区分配置读取、类型、大小、文件缺失及载荷问题，供响应状态和翻译使用。
const (
	ErrCodeBrandingConfigUnavailable  = "branding_config_unavailable"
	ErrCodeBrandingLogoInvalidType    = "branding_logo_invalid_type"
	ErrCodeBrandingFaviconInvalidType = "branding_favicon_invalid_type"
	ErrCodeBrandingInternalError      = "branding_internal_error"
	ErrCodeBrandingInvalidPayload     = "branding_invalid_payload"
	ErrCodeBrandingLogoTooLarge       = "branding_logo_too_large"
	ErrCodeBrandingLogoMissing        = "branding_logo_missing"
	ErrCodeBrandingFaviconTooLarge    = "branding_favicon_too_large"
	ErrCodeBrandingFaviconMissing     = "branding_favicon_missing"
)

// respondWithBrandingStatusError 交给公共错误编码器输出明确状态与翻译参数。
func respondWithBrandingStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithBrandingError 依次处理本地化错误和字段错误集合，其他失败使用固定 500 消息。
func respondWithBrandingError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithBrandingStatusError(c, brandingErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	var vErr *sharederrors.ValidationErrors
	if errors.As(err, &vErr) {
		httpinternal.RespondValidationErrors(c, vErr)
		return
	}

	respondWithBrandingStatusError(c, http.StatusInternalServerError, ErrCodeBrandingInternalError, "Branding request failed", "branding request failed")
}

// brandingErrorStatus 将类型、载荷及缺文件归为 400，大小超限归为 413，其他归为 500。
func brandingErrorStatus(code string) int {
	switch code {
	case ErrCodeBrandingLogoInvalidType, ErrCodeBrandingFaviconInvalidType,
		ErrCodeBrandingInvalidPayload, ErrCodeBrandingLogoMissing, ErrCodeBrandingFaviconMissing:
		return http.StatusBadRequest
	case ErrCodeBrandingLogoTooLarge, ErrCodeBrandingFaviconTooLarge:
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusInternalServerError
	}
}
