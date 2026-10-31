// 本文件为用户上传文件设置浏览器响应头，使用扩展名选择预览或下载。
// 它不读取文件正文，调用方仍负责查找资源、权限判断和实际发送文件。
package security

import (
	"mime"
	"net/http"
	"path"
	"strings"
)

// inlineFileExts 指定以 inline 方式发送的图片、音视频和 PDF 扩展名。
// 扩展名不在集合中时要求下载，包括 HTML、SVG、XML、未知扩展名与无扩展名文件。
// 选择依据是文件名而非内容检测，内容类型仍由文件响应的其他处理决定。
var inlineFileExts = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".avif": {}, ".bmp": {}, ".ico": {},
	".pdf": {},
	".mp4": {}, ".webm": {}, ".ogv": {}, ".mov": {}, ".m4v": {},
	".mp3": {}, ".wav": {}, ".ogg": {}, ".oga": {}, ".m4a": {}, ".flac": {}, ".aac": {}, ".opus": {},
}

// SetUntrustedFileHeaders 为用户文件响应设置禁止类型嗅探及内容展示方式。
// 除 PDF 外写入无授权项的 CSP sandbox，约束被浏览器当作文档加载的内容。
// PDF 分支不写该头，也不删除调用方已经设置的同名头。
// 文件名取路径末段，使用标准库编码进 Content-Disposition；编码失败时只保留展示类型。
// 函数只操作响应头，不验证文件内容，也不负责实际发送文件。
func SetUntrustedFileHeaders(h http.Header, name string) {
	h.Set("X-Content-Type-Options", "nosniff")

	// 扩展名转小写后比较，名称大小写不会改变 PDF 豁免和内联集合的判断。
	ext := strings.ToLower(path.Ext(name))
	if ext != ".pdf" {
		h.Set("Content-Security-Policy", "sandbox")
	}

	disposition := "attachment"
	if _, ok := inlineFileExts[ext]; ok {
		disposition = "inline"
	}
	if v := mime.FormatMediaType(disposition, map[string]string{"filename": path.Base(name)}); v != "" {
		disposition = v
	}
	h.Set("Content-Disposition", disposition)
}
