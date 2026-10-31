// 本文件在路由处理结束后翻译约定的 JSON 诊断字段。
// 页面正文和任意业务字符串不做全量递归替换，只有明确列出的字段进入翻译流程。
package http

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/localization"
	"github.com/gin-gonic/gin"
)

// localizedWriter 根据写入当时的 Content-Type 判断是否缓冲，
// 只有包含 application/json 的响应进入缓冲，其他字节直接写给原响应器。
type localizedWriter struct {
	gin.ResponseWriter
	buffer bytes.Buffer
}

// Write 不改变状态码或响应头，仅拦截 JSON 正文，返回缓冲器或底层写入器的结果。
func (w *localizedWriter) Write(data []byte) (int, error) {
	if strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		return w.buffer.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

// WriteString 通过相同字节入口写入，避免字符串写入绕过 JSON 缓冲。
func (w *localizedWriter) WriteString(value string) (int, error) { return w.Write([]byte(value)) }

// localizeJSON 为每个请求包装响应器，设置事件地址直接跳过包装以保留流式传输。
// 路由结束后尝试解析与重新编码缓冲正文，失败保留原始字节，
// 删除原 Content-Length 避免翻译后长度不符，底层最终写入错误不在此返回。
func localizeJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSettingsStream(c.Request.URL.Path) {
			c.Next()
			return
		}
		writer := &localizedWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()
		if writer.buffer.Len() == 0 {
			return
		}
		// 没有 JSON 正文字节时直接结束，不额外生成响应。
		data := writer.buffer.Bytes()
		var value any
		if json.Unmarshal(data, &value) == nil {
			translateResponse(value, writer.Status())
			if encoded, err := json.Marshal(value); err == nil {
				data = encoded
			}
		}
		writer.Header().Del("Content-Length")
		_, _ = writer.ResponseWriter.Write(data)
	}
}

// translateResponse 只处理对象值。结构化 error 优先按 template 与字符串参数翻译，
// 找不到模板且 message 为空时使用状态码兜底，有原消息则保留。
// 普通 message/error、字段错误及导入计划的指定数组分别处理，不扫描任意对象键。
func translateResponse(value any, status int) {
	root, ok := value.(map[string]any)
	if !ok {
		return
	}
	if detail, ok := root["error"].(map[string]any); ok {
		template, _ := detail["template"].(string)
		message, _ := detail["message"].(string)
		var args []string
		// JSON 参数中只收集字符串，其他类型不传入模板格式化。
		if values, ok := detail["args"].([]any); ok {
			for _, v := range values {
				if s, ok := v.(string); ok {
					args = append(args, s)
				}
			}
		}
		translated := localization.Message(template, args...)
		if translated != template {
			detail["message"] = translated
		} else if message == "" {
			detail["message"] = localization.Fallback(status)
		}
	}
	// validation_error 是字段校验协议标识，不能当作自然语言替换。
	for _, key := range []string{"message", "error"} {
		if message, ok := root[key].(string); ok && (key != "error" || message != "validation_error") {
			translated := localization.Message(strings.ToLower(message))
			if translated != strings.ToLower(message) {
				root[key] = translated
			} else if message == "" {
				root[key] = localization.Fallback(status)
			}
		}
	}
	if fields, ok := root["fields"].([]any); ok {
		for _, field := range fields {
			if v, ok := field.(map[string]any); ok {
				translateResponse(v, status)
			}
		}
	}
	// 导入计划和校验诊断数组只翻译字符串元素，空文本按状态四百生成兜底消息。
	for _, key := range []string{"errors", "notes", "conflicts"} {
		if values, ok := root[key].([]any); ok {
			for i, raw := range values {
				if message, ok := raw.(string); ok {
					translated := localization.Message(strings.ToLower(message))
					if translated != strings.ToLower(message) {
						values[i] = translated
					} else if message == "" {
						values[i] = localization.Fallback(400)
					}
				}
			}
		}
	}
	// items 与 plan 可以是对象或对象数组，沿这些约定结构递归处理诊断字段。
	for _, key := range []string{"items", "plan"} {
		if values, ok := root[key].([]any); ok {
			for _, item := range values {
				translateResponse(item, status)
			}
		}
		if value, ok := root[key].(map[string]any); ok {
			translateResponse(value, status)
		}
	}
}
