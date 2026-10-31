package avatar

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// 头像错误代码区分类型、解码、写入、删除及未分类失败，供状态与消息翻译使用。
const (
	ErrCodeAvatarInvalidType   = "avatar_invalid_type"
	ErrCodeAvatarDecodeFailed  = "avatar_decode_failed"
	ErrCodeAvatarUploadFailed  = "avatar_upload_failed"
	ErrCodeAvatarDeleteFailed  = "avatar_delete_failed"
	ErrCodeAvatarInternalError = "avatar_internal_error"
)

// respondWithAvatarStatusError 将指定状态和本地化字段交给公共错误编码器。
func respondWithAvatarStatusError(c *gin.Context, status int, code, message, template string, args ...string) {
	httpinternal.RespondError(c, status, code, message, template, args...)
}

// respondWithAvatarError 按领域代码映射状态，未知错误使用固定内部失败内容。
func respondWithAvatarError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithAvatarStatusError(c, avatarErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template, loc.Args...)
		return
	}

	respondWithAvatarStatusError(c, http.StatusInternalServerError, ErrCodeAvatarInternalError, "Avatar request failed", "avatar request failed")
}

// avatarErrorStatus 将图片类型与解码错误归为 400，其余已知或未知代码归为 500。
func avatarErrorStatus(code string) int {
	switch code {
	case ErrCodeAvatarInvalidType, ErrCodeAvatarDecodeFailed:
		return http.StatusBadRequest
	case ErrCodeAvatarUploadFailed, ErrCodeAvatarDeleteFailed, ErrCodeAvatarInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
