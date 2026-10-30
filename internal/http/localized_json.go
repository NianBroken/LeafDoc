package http

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/localization"
	"github.com/gin-gonic/gin"
)

// localizedWriter buffers JSON only; static files, downloads and event streams pass through.
type localizedWriter struct {
	gin.ResponseWriter
	buffer bytes.Buffer
}

func (w *localizedWriter) Write(data []byte) (int, error) {
	if strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		return w.buffer.Write(data)
	}
	return w.ResponseWriter.Write(data)
}
func (w *localizedWriter) WriteString(value string) (int, error) { return w.Write([]byte(value)) }
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
func translateResponse(value any, status int) {
	root, ok := value.(map[string]any)
	if !ok {
		return
	}
	if detail, ok := root["error"].(map[string]any); ok {
		template, _ := detail["template"].(string)
		message, _ := detail["message"].(string)
		var args []string
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
	for _, key := range []string{"message", "error"} {
		if message, ok := root[key].(string); ok && !(key == "error" && message == "validation_error") {
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
	// Import plans and validation responses carry system-generated diagnostics as arrays.
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
func containsHan(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}
