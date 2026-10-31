// Package htmlutil 提供普通文本转义和指定标记替换，供生成 HTML 片段的代码调用。
package htmlutil

import (
	"html"
	"strings"
)

// AllowedMarker 指定转义后文本中的一个标记及其替换片段。
// HTML 不再经过转义，调用方必须明确掌握其内容。
type AllowedMarker struct {
	Marker string
	HTML   string
}

// EscapeText 将 HTML 特殊字符转换为实体，返回可按普通文本展示的字符串。
// 处理直接交给标准库 html.EscapeString，不解析标签或移除文本。
func EscapeText(text string) string {
	return html.EscapeString(text)
}

// EscapeTextWithAllowedMarkers 先转义全文，再按参数顺序替换所有非空标记。
// 后续标记还会作用于前一次替换的结果，因此标记顺序会影响重叠内容。
// 此函数不检查替换片段是否安全，也不限制它必须是某种标签。
func EscapeTextWithAllowedMarkers(text string, markers ...AllowedMarker) string {
	escaped := EscapeText(text)
	for _, marker := range markers {
		// 空标记不表示插入位置，直接跳过以避免在每个字符间插入片段。
		if marker.Marker == "" {
			continue
		}
		escaped = strings.ReplaceAll(escaped, marker.Marker, marker.HTML)
	}
	return escaped
}
