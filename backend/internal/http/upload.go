// 本文件集中处理单文件 multipart 上传的请求体限额及文件字段提取。
package http

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrUploadTooLarge 表示 multipart 解析失败，本入口将超限及其他解析错误统一映射到此值。
var ErrUploadTooLarge = errors.New("upload too large")

// ErrUploadMissingFile 表示文件字段获取失败，既包括字段不存在，也包括 FormFile 的其他错误。
var ErrUploadMissingFile = errors.New("missing uploaded file")

// ParseUploadedFile 用 MaxBytesReader 限制整个请求体，再解析 multipart 并打开指定字段。
// maxSize 也作为解析器的内存参数，限制对象包含 multipart 边界等开销，不只文件内容。
// 返回文件由调用方关闭，表单临时文件清理由请求生命周期负责；错误只返回统一分类，
// 不保留解析器的底层原因，此入口不自行向客户端写入 JSON。
func ParseUploadedFile(c *gin.Context, maxSize int64, fieldName string) (multipart.File, *multipart.FileHeader, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
	if err := c.Request.ParseMultipartForm(maxSize); err != nil {
		return nil, nil, ErrUploadTooLarge
	}

	file, header, err := c.Request.FormFile(fieldName)
	if err != nil {
		return nil, nil, ErrUploadMissingFile
	}

	return file, header, nil
}
