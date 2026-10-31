// 本文件定义业务路由共用的错误 JSON 结构，翻译由响应中间件按模板另行完成。
package http

import (
	"net/http"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// ErrorDetail 分开保存错误代码、现有消息、翻译模板及可选参数。
// 空参数切片省略，其他字段按传入值返回，不在结构中推断 HTTP 状态。
type ErrorDetail struct {
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Template string   `json:"template"`
	Args     []string `json:"args,omitempty"`
}

// ErrorResponse 把结构化详情放入 error 字段，区别于只含字符串的错误响应。
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// RespondError 按指定状态码写 JSON，并复制参数切片，避免响应持有调用方的切片元素区间。
// 此函数不调用 Abort，是否停止后续处理由调用路由决定。
func RespondError(c *gin.Context, status int, code, message, template string, args ...string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:     code,
			Message:  message,
			Template: template,
			Args:     append([]string(nil), args...),
		},
	})
}

// RespondValidationErrors 固定返回四百状态和 validation_error 标识，
// 字段详情直接取自非空校验对象，不在此逐项翻译或复制字段列表。
func RespondValidationErrors(c *gin.Context, fields *sharederrors.ValidationErrors) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "fields": fields.Errors})
}
