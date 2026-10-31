package instancesettings

import (
	"net/http"

	"github.com/NianBroken/LeafDoc/internal/core/publicaccess"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	"github.com/gin-gonic/gin"
)

const (
	// ErrCodeInvalidPayload 表示请求体无法解析或缺少处理器要求的字段。
	ErrCodeInvalidPayload = "instance_settings_invalid_payload"
	// ErrCodeInternal 用于不能归类为已知本地化错误的内部失败。
	ErrCodeInternal = "instance_settings_internal_error"
)

// respondWithStatusError 将状态、错误代码及翻译信息交给 HTTP 层统一编码。
func respondWithStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithError 对本地化错误保留消息和参数，其他错误使用固定 500 回退内容。
func respondWithError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithStatusError(c, statusForCode(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}
	respondWithStatusError(c, http.StatusInternalServerError, ErrCodeInternal,
		"Instance settings request failed", "instance settings request failed")
}

// statusForCode 将环境管理冲突映射为 409，载荷或目录模式错误映射为 400。
// 其余代码按内部失败处理，不把所有领域错误一律视为客户端输入问题。
func statusForCode(code string) int {
	switch code {
	case publicaccess.ErrCodeEnvManaged:
		return http.StatusConflict
	case ErrCodeInvalidPayload, "toc_display_invalid_mode":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
