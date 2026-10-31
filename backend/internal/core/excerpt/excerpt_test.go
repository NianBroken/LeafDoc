// 本文件核验摘要与搜索文本的差异，包括代码保留、围栏识别和行内语法清理。
// 所有样例只调用文本转换函数，不启动 Markdown 页面或浏览器。
package excerpt

import (
	"strings"
	"testing"
)

// 前置字段被剥离，摘要只取实际正文。
func TestFromContent_PlainText(t *testing.T) {
	raw := "---\ntitle: Hello\n---\n\nThis is the page body."
	got := FromContent(raw)
	if got != "This is the page body." {
		t.Errorf("got %q", got)
	}
}

// 没有前置字段时，普通文本原样进入摘要提取流程。
func TestFromContent_NoFrontmatter(t *testing.T) {
	raw := "Just plain content here."
	got := FromContent(raw)
	if got != "Just plain content here." {
		t.Errorf("got %q", got)
	}
}

// 只有前置字段且正文为空时得到空摘要。
func TestFromContent_EmptyBody(t *testing.T) {
	raw := "---\ntitle: Hello\n---\n\n"
	got := FromContent(raw)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// 空输入对应空结果，不生成替代内容。
func TestFromContent_EmptyContent(t *testing.T) {
	got := FromContent("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// 显示摘要去除代码块，同时保留两侧正文。
func TestFromContent_StripsFencedCode(t *testing.T) {
	raw := "---\ntitle: T\n---\n\nBefore.\n\n```go\nfunc main() {}\n```\n\nAfter."
	got := FromContent(raw)
	if strings.Contains(got, "func main") {
		t.Errorf("excerpt should not contain fenced code, got %q", got)
	}
	if !strings.Contains(got, "Before.") || !strings.Contains(got, "After.") {
		t.Errorf("excerpt should contain surrounding text, got %q", got)
	}
}

// 搜索文本保留普通代码块内容，使代码词与正文词都能进入检索输入。
func TestPlainTextForSearch_IncludesCodeBlockContent(t *testing.T) {
	body := "Before.\n\n```go\nfunc main() {}\n```\n\nAfter."
	got := PlainTextForSearch(body)
	if !strings.Contains(got, "func main") {
		t.Errorf("search text should contain fenced code content, got %q", got)
	}
	if !strings.Contains(got, "Before.") || !strings.Contains(got, "After.") {
		t.Errorf("search text should contain surrounding text, got %q", got)
	}
}

// 标题文字保留，井号标记不显示。
func TestFromContent_StripsMarkdownHeadings(t *testing.T) {
	raw := "---\ntitle: T\n---\n\n# Heading One\n\nSome body text."
	got := FromContent(raw)
	if strings.Contains(got, "#") {
		t.Errorf("excerpt should not contain # heading markers, got %q", got)
	}
	if !strings.Contains(got, "Heading One") {
		t.Errorf("heading text should be preserved, got %q", got)
	}
}

// 图片转换保留替代文字而移除目标地址和图片语法。
func TestFromContent_StripsImageSyntax(t *testing.T) {
	raw := "---\ntitle: T\n---\n\n![alt text](image.png) Some text."
	got := FromContent(raw)
	if strings.Contains(got, "![") || strings.Contains(got, "image.png") {
		t.Errorf("image syntax should be stripped, got %q", got)
	}
	if !strings.Contains(got, "alt text") {
		t.Errorf("alt text should be preserved, got %q", got)
	}
}

// 链接转换保留文字而移除 URL。
func TestFromContent_StripsLinkSyntax(t *testing.T) {
	raw := "---\ntitle: T\n---\n\n[Click here](https://example.com) for more."
	got := FromContent(raw)
	if strings.Contains(got, "https://example.com") || strings.Contains(got, "](") {
		t.Errorf("link URL should be stripped, got %q", got)
	}
	if !strings.Contains(got, "Click here") {
		t.Errorf("link text should be preserved, got %q", got)
	}
}

// 长正文检查截断标记及字符数量范围，不依赖恰好截在某个单词之后。
func TestFromContent_TruncatesLongContent(t *testing.T) {
	body := strings.Repeat("word ", 200)
	raw := "---\ntitle: T\n---\n\n" + body
	got := FromContent(raw)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("long content should be truncated with ellipsis, got %q", got)
	}
	runes := []rune(got)
	if len(runes) > MaxRunes+10 {
		t.Errorf("excerpt too long: %d runes", len(runes))
	}
}

// 短正文不添加省略号。
func TestFromContent_ShortContentNotTruncated(t *testing.T) {
	raw := "---\ntitle: T\n---\n\nShort body."
	got := FromContent(raw)
	if strings.HasSuffix(got, "...") {
		t.Errorf("short content should not be truncated, got %q", got)
	}
}

// 段落换行合并为空格，检查结果不含换行和连续双空格。
func TestFromContent_CollapsesWhitespace(t *testing.T) {
	raw := "---\ntitle: T\n---\n\nLine one.\n\nLine two.\n\nLine three."
	got := FromContent(raw)
	if strings.Contains(got, "\n") {
		t.Errorf("excerpt should have no newlines, got %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("excerpt should have no double spaces, got %q", got)
	}
}

// 显示摘要移除 HTML 标记，本断言检查指定标签已消失。
func TestFromContent_StripsHTMLTags(t *testing.T) {
	raw := "---\ntitle: T\n---\n\n<strong>Bold</strong> text."
	got := FromContent(raw)
	if strings.Contains(got, "<strong>") || strings.Contains(got, "</strong>") {
		t.Errorf("HTML tags should be stripped, got %q", got)
	}
}

// 强调符号清理后，原文字和标点仍连续保留。
func TestFromContent_StripsMarkdownEmphasisMarkers(t *testing.T) {
	raw := "---\ntitle: T\n---\n\nLeafDoc **fett** und _kursiv_."
	got := FromContent(raw)
	if strings.Contains(got, "**") || strings.Contains(got, "_") {
		t.Errorf("markdown emphasis markers should be stripped, got %q", got)
	}
	if !strings.Contains(got, "LeafDoc fett und kursiv.") {
		t.Errorf("expected plain text excerpt, got %q", got)
	}
}

// 提示块去除三冒号围栏，但内置与自定义标签、块内正文都保留。
func TestNormalizeMarkdownBody_KeepsLabelsAndRemovesShoutoutFenceSyntax(t *testing.T) {
	body := strings.Join([]string{
		"::: info",
		"Helpful details.",
		":::",
		"",
		"::: custom-banner",
		"Custom text.",
		":::",
	}, "\n")

	got := NormalizeMarkdownBody(body)

	if strings.Contains(got, ":::") {
		t.Fatalf("expected fence syntax to be removed, got %q", got)
	}
	for _, want := range []string{"info", "custom-banner", "Helpful details.", "Custom text."} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected normalized content to contain %q, got %q", want, got)
		}
	}
}

// 代码围栏内的三冒号按字面内容保留，整个代码块与输入逐字一致。
func TestNormalizeMarkdownBody_IgnoresCodeFences(t *testing.T) {
	body := strings.Join([]string{
		"```md",
		"::: info",
		"literal",
		":::",
		"```",
	}, "\n")

	got := NormalizeMarkdownBody(body)

	if got != body {
		t.Fatalf("expected fenced code block to stay unchanged, got %q", got)
	}
}

// 表驱动样例分别核验普通双链、别名、图片嵌入及路径末段显示规则。
func TestPlainTextFromMarkdown_StripsWikiLinks(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"plain wikilink", "See [[Some Page]] for details.", "See Some Page for details."},
		{"aliased wikilink", "See [[Some Page|this page]] for details.", "See this page for details."},
		{"image embed", "![[image.png]] Some text.", "Some text."},
		{"path-hinted wikilink shows last segment", "See [[reference/endpoints]] for details.", "See endpoints for details."},
		{"path-hinted aliased wikilink shows alias", "See [[reference/endpoints|API]] for details.", "See API for details."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PlainTextFromMarkdown(tc.body)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// 代码块内的双链按代码文字处理，页面名和别名均进入搜索文本。
func TestPlainTextForSearch_WikiLinkAliasInCodeBlockPreservesPageName(t *testing.T) {
	body := "```\n[[SomePage|Display Text]]\n```"
	got := PlainTextForSearch(body)
	if !strings.Contains(got, "SomePage") {
		t.Errorf("page name inside code block should be searchable, got %q", got)
	}
	if !strings.Contains(got, "Display") {
		t.Errorf("alias text inside code block should be searchable, got %q", got)
	}
}

// 代码中的图片嵌入不按正文图片删除，文件名仍可检索。
func TestPlainTextForSearch_WikiImageInCodeBlockPreservesFilename(t *testing.T) {
	body := "```\n![[screenshot.png]]\n```"
	got := PlainTextForSearch(body)
	if !strings.Contains(got, "screenshot.png") {
		t.Errorf("image filename inside code block should be searchable, got %q", got)
	}
}

// 围栏外的双链仍按正文规则提取别名，不保留被别名替代的页面名。
func TestPlainTextForSearch_InlinePatternsProseMayRunNormally(t *testing.T) {
	body := "See [[ProseLink|prose alias]] and ![[icon.png]] outside code."
	got := PlainTextForSearch(body)
	if !strings.Contains(got, "prose alias") {
		t.Errorf("wiki link alias outside code block should be extracted, got %q", got)
	}
	if strings.Contains(got, "ProseLink") {
		t.Errorf("wiki link page name outside code block should not appear verbatim, got %q", got)
	}
}

// 同一输入混合正文与代码时，正文提取别名，代码保留页面名。
func TestPlainTextForSearch_MixedCodeAndProsePatterns(t *testing.T) {
	body := "See [[ProseLink|prose]] for info.\n\n```\n[[CodePage|code alias]]\n```"
	got := PlainTextForSearch(body)
	if !strings.Contains(got, "prose") {
		t.Errorf("prose wiki link alias should be extracted, got %q", got)
	}
	if !strings.Contains(got, "CodePage") {
		t.Errorf("page name inside code block should be searchable, got %q", got)
	}
}

// Mermaid 图表块不进入搜索文本，周围段落仍可检索。
func TestPlainTextForSearch_ExcludesMermaidBlocks(t *testing.T) {
	body := "Some prose.\n\n```mermaid\ngraph TD\n    A[Login] --> B{Auth ok?}\n```\n\nMore prose."
	got := PlainTextForSearch(body)
	if strings.Contains(got, "graph") || strings.Contains(got, "-->") {
		t.Errorf("mermaid diagram syntax should not be in search index, got %q", got)
	}
	if !strings.Contains(got, "Some prose.") || !strings.Contains(got, "More prose.") {
		t.Errorf("surrounding prose should still be indexed, got %q", got)
	}
}

// 显示摘要同样排除 Mermaid 图表语法。
func TestPlainTextFromMarkdown_ExcludesMermaidBlocks(t *testing.T) {
	body := "Some prose.\n\n```mermaid\ngraph TD\n    A[Login] --> B{Auth ok?}\n```\n\nMore prose."
	got := PlainTextFromMarkdown(body)
	if strings.Contains(got, "graph") || strings.Contains(got, "-->") {
		t.Errorf("mermaid diagram syntax should not appear in display excerpt, got %q", got)
	}
}

// 引用块前缀下的代码围栏仍保护块内图片语法，文件名保留在搜索文本中。
func TestPlainTextForSearch_WikiImageInBlockQuotedCodeBlock(t *testing.T) {
	body := "> ```\n> ![[screenshot.png]]\n> ```"
	got := PlainTextForSearch(body)
	if !strings.Contains(got, "screenshot.png") {
		t.Errorf("image filename in block-quoted code should be searchable, got %q", got)
	}
}

// 开始围栏这一行本身不执行行内图片替换，后续代码文字也保持原样。
func TestApplyInlinePatternsOutsideFences_FenceOpenLineNotProcessed(t *testing.T) {
	input := "```![[img.png]]\ncurl\n```"
	got := applyInlinePatternsOutsideFences(input)
	lines := strings.Split(got, "\n")
	if lines[0] != "```![[img.png]]" {
		t.Errorf("fence-open line should not be processed by patterns, got %q", lines[0])
	}
	if lines[1] != "curl" {
		t.Errorf("code content should be preserved verbatim, got %q", lines[1])
	}
}

// FromBody 处理无前置字段的正文，组合清理 HTML 与普通链接并保留可读文字。
func TestFromBody_StripsHTMLAndMarkdown(t *testing.T) {
	body := "<p><strong>Bold</strong> [link](https://example.com)</p>"
	got := FromBody(body)
	if got != "Bold link" {
		t.Errorf("got %q", got)
	}
}
