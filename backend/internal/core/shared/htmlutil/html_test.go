// 本文件检查文本转义与显式标记替换，所有断言直接比较最终字符串。
package htmlutil

import "testing"

// TestEscapeText 以脚本标签和双引号确认 HTML 特殊字符被转成实体。
func TestEscapeText(t *testing.T) {
	got := EscapeText(`<script>alert("x")</script>`)
	want := "&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;"
	if got != want {
		t.Fatalf("escaped = %q, want %q", got, want)
	}
}

// TestEscapeTextWithAllowedMarkers 验证原始脚本内容保持转义，指定控制字符转为粗体标签。
// 这一断言同时检查普通文本与允许片段在同一个输出中的处理差异。
func TestEscapeTextWithAllowedMarkers(t *testing.T) {
	got := EscapeTextWithAllowedMarkers(
		`<script>alert(1)</script> `+"\u0002"+`Search`+"\u0003",
		AllowedMarker{Marker: "\u0002", HTML: "<b>"},
		AllowedMarker{Marker: "\u0003", HTML: "</b>"},
	)

	if got != "&lt;script&gt;alert(1)&lt;/script&gt; <b>Search</b>" {
		t.Fatalf("escaped = %q", got)
	}
}

// TestEscapeTextWithAllowedMarkers_SkipsEmptyMarkers 确认空标记不会向已转义文本插入内容。
func TestEscapeTextWithAllowedMarkers_SkipsEmptyMarkers(t *testing.T) {
	got := EscapeTextWithAllowedMarkers(`<b>title</b>`, AllowedMarker{})
	want := "&lt;b&gt;title&lt;/b&gt;"
	if got != want {
		t.Fatalf("escaped = %q, want %q", got, want)
	}
}
