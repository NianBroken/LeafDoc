// 本文件验证搜索索引的标题文本提取，不连接 MySQL 或执行全文搜索。
// 断言以文字保留和格式标记移除为主，不限定标题之间的具体空白格式。
package search

import (
	"strings"
	"testing"
)

// 单个一级标题应将标题文字写入检索文本。
func TestExtractHeadings_SingleH1(t *testing.T) {
	got := extractHeadings("# Hello World")
	if !strings.Contains(got, "Hello World") {
		t.Errorf("expected heading text, got %q", got)
	}
}

// 一级到三级标题均应参与提取，用例检查每个文字存在，不比较完整排列字符串。
func TestExtractHeadings_MultipleHeadings(t *testing.T) {
	input := "# First\n## Second\n### Third"
	got := extractHeadings(input)
	for _, want := range []string{"First", "Second", "Third"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in result, got %q", want, got)
		}
	}
}

// 标题中的粗体和斜体标记应移除，但相应文字与普通标题文字必须保留。
func TestExtractHeadings_InlineFormatting(t *testing.T) {
	got := extractHeadings("## **Bold** and _italic_ heading")
	if strings.Contains(got, "**") || strings.Contains(got, "_") {
		t.Errorf("expected formatting markers stripped, got %q", got)
	}
	if !strings.Contains(got, "Bold") || !strings.Contains(got, "italic") || !strings.Contains(got, "heading") {
		t.Errorf("expected heading words preserved, got %q", got)
	}
}

// 正文段落不进入 headings 字段，标题文字仍然可用于搜索权重计算。
func TestExtractHeadings_IgnoresBodyText(t *testing.T) {
	got := extractHeadings("# Title\n\nSome body paragraph that should not appear.")
	if strings.Contains(got, "body paragraph") {
		t.Errorf("body text should not appear in headings, got %q", got)
	}
	if !strings.Contains(got, "Title") {
		t.Errorf("expected heading text, got %q", got)
	}
}

// 只有普通段落时返回空文本，不把整篇正文当作标题字段。
func TestExtractHeadings_NoHeadings(t *testing.T) {
	got := extractHeadings("Just plain text without any heading.")
	if got != "" {
		t.Errorf("expected empty result for no headings, got %q", got)
	}
}

// 空正文无需特殊占位值，标题提取结果也为空。
func TestExtractHeadings_EmptyInput(t *testing.T) {
	got := extractHeadings("")
	if got != "" {
		t.Errorf("expected empty result for empty input, got %q", got)
	}
}

// 标题包含行内代码时，不保留反引号，代码前后的标题文字仍须存在。
// 本用例没有断言代码单词本身是否保留，不扩大其验证结论。
func TestExtractHeadings_HeadingWithCodeSpan(t *testing.T) {
	got := extractHeadings("## Heading `code` here")
	if strings.Contains(got, "`") {
		t.Errorf("expected backticks stripped, got %q", got)
	}
	if !strings.Contains(got, "Heading") || !strings.Contains(got, "here") {
		t.Errorf("expected heading words preserved, got %q", got)
	}
}
