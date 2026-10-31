package links

import (
	"net/http"

	httpinternal "github.com/NianBroken/LeafDoc/internal/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// 链接领域代码分别表示页面缺失、服务未配置与内部查询失败。
const (
	ErrCodeLinkPageNotFound  = "link_page_not_found"
	ErrCodeLinkUnavailable   = "link_service_unavailable"
	ErrCodeLinkInternalError = "link_internal_error"
)

// respondWithLinkStatusError 输出代码、消息与模板，此入口没有可变翻译参数。
func respondWithLinkStatusError(c *gin.Context, status int, code, message, template string) {
	httpinternal.RespondError(c, status, code, message, template)
}

// respondWithLinkError 对本地化错误保留代码、消息和模板，不转交 loc.Args。
// 普通错误使用固定 500 响应，避免将底层数据库错误直接发送给客户端。
func respondWithLinkError(c *gin.Context, err error) {
	if loc, ok := sharederrors.AsLocalizedError(err); ok {
		respondWithLinkStatusError(c, linkErrorStatus(loc.Code), loc.Code, loc.Message, loc.Template)
		return
	}

	respondWithLinkStatusError(c, http.StatusInternalServerError, ErrCodeLinkInternalError, "Failed to load link status", "failed to load link status")
}

// linkErrorStatus 将页面缺失映射为 404、服务不可用映射为 503，其余按内部失败处理。
func linkErrorStatus(code string) int {
	switch code {
	case ErrCodeLinkPageNotFound:
		return http.StatusNotFound
	case ErrCodeLinkUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
