// 本文件通过标签模块公开入口核验摘要清理，确保委托调用保留公共提取规则。
package tags

import (
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/excerpt"
)

// 验证前置标题不进入摘要，实际返回的是分隔线之后的正文。
func TestExtractExcerptFromContent_PlainText(t *testing.T) {
	raw := "---\ntitle: Hello\n---\n\nThis is the page body."
	got := ExtractExcerptFromContent(raw)
	if got != "This is the page body." {
		t.Errorf("got %q", got)
	}
}

// 无前置字段的普通文字直接作为正文处理。
func TestExtractExcerptFromContent_NoFrontmatter(t *testing.T) {
	raw := "Just plain content here."
	got := ExtractExcerptFromContent(raw)
	if got != "Just plain content here." {
		t.Errorf("got %q", got)
	}
}

// 只有前置字段而正文为空时返回空串，不把标题当作正文回退值。
func TestExtractExcerptFromContent_EmptyBody(t *testing.T) {
	raw := "---\ntitle: Hello\n---\n\n"
	got := ExtractExcerptFromContent(raw)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// 空输入不生成占位文字。
func TestExtractExcerptFromContent_EmptyContent(t *testing.T) {
	got := ExtractExcerptFromContent("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// 摘要移除围栏代码内容，同时保留代码块前后的普通段落。
func TestExtractExcerptFromContent_StripsFencedCode(t *testing.T) {
	raw := "---\ntitle: T\n---\n\nBefore.\n\n```go\nfunc main() {}\n```\n\nAfter."
	got := ExtractExcerptFromContent(raw)
	if strings.Contains(got, "func main") {
		t.Errorf("excerpt should not contain fenced code, got %q", got)
	}
	if !strings.Contains(got, "Before.") || !strings.Contains(got, "After.") {
		t.Errorf("excerpt should contain surrounding text, got %q", got)
	}
}

// 标题标记被清理，标题本身的文字仍属于可显示摘要。
func TestExtractExcerptFromContent_StripsMarkdownHeadings(t *testing.T) {
	raw := "---\ntitle: T\n---\n\n# Heading One\n\nSome body text."
	got := ExtractExcerptFromContent(raw)
	if strings.Contains(got, "#") {
		t.Errorf("excerpt should not contain # heading markers, got %q", got)
	}
	if !strings.Contains(got, "Heading One") {
		t.Errorf("heading text should be preserved, got %q", got)
	}
}

// 图片地址与语法被移除，替代文字仍被保留。
func TestExtractExcerptFromContent_StripsImageSyntax(t *testing.T) {
	raw := "---\ntitle: T\n---\n\n![alt text](image.png) Some text."
	got := ExtractExcerptFromContent(raw)
	if strings.Contains(got, "![") || strings.Contains(got, "image.png") {
		t.Errorf("image syntax should be stripped, got %q", got)
	}
	if !strings.Contains(got, "alt text") {
		t.Errorf("alt text should be preserved, got %q", got)
	}
}

// 普通链接保留可读文字，目标地址不进入显示摘要。
func TestExtractExcerptFromContent_StripsLinkSyntax(t *testing.T) {
	raw := "---\ntitle: T\n---\n\n[Click here](https://example.com) for more."
	got := ExtractExcerptFromContent(raw)
	if strings.Contains(got, "https://example.com") || strings.Contains(got, "](") {
		t.Errorf("link URL should be stripped, got %q", got)
	}
	if !strings.Contains(got, "Click here") {
		t.Errorf("link text should be preserved, got %q", got)
	}
}

// 构造远超摘要长度的正文，检查省略号和按字符计算的结果上限。
func TestExtractExcerptFromContent_TruncatesLongContent(t *testing.T) {
	body := strings.Repeat("word ", 200)
	raw := "---\ntitle: T\n---\n\n" + body
	got := ExtractExcerptFromContent(raw)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("long content should be truncated with ellipsis, got %q", got)
	}
	runes := []rune(got)
	if len(runes) > excerpt.MaxRunes+10 {
		t.Errorf("excerpt too long: %d runes", len(runes))
	}
}

// 短正文不带截断标记。
func TestExtractExcerptFromContent_ShortContentNotTruncated(t *testing.T) {
	raw := "---\ntitle: T\n---\n\nShort body."
	got := ExtractExcerptFromContent(raw)
	if strings.HasSuffix(got, "...") {
		t.Errorf("short content should not be truncated, got %q", got)
	}
}

// 多段文字汇合后不保留换行或连续双空格。
func TestExtractExcerptFromContent_CollapsesWhitespace(t *testing.T) {
	raw := "---\ntitle: T\n---\n\nLine one.\n\nLine two.\n\nLine three."
	got := ExtractExcerptFromContent(raw)
	if strings.Contains(got, "\n") {
		t.Errorf("excerpt should have no newlines, got %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("excerpt should have no double spaces, got %q", got)
	}
}

// HTML 标签语法不出现在显示结果中，本断言专门检查开始和结束标签。
func TestExtractExcerptFromContent_StripsHTMLTags(t *testing.T) {
	raw := "---\ntitle: T\n---\n\n<strong>Bold</strong> text."
	got := ExtractExcerptFromContent(raw)
	if strings.Contains(got, "<strong>") || strings.Contains(got, "</strong>") {
		t.Errorf("HTML tags should be stripped, got %q", got)
	}
}
