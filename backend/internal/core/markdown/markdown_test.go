// 本文件验证带路径 Markdown 对象的标题优先级、字段保留和实际文件写回。
// 所有落盘案例使用测试临时目录，Windows 路径案例只在内存中提取文件名。
package markdown

import (
	"os"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/testutil"
)

// TestPlanner_extractTitleFromMDFile_FrontmatterTitleWins 检查普通 title 元信息优先于正文标题。
func TestPlanner_extractTitleFromMDFile_FrontmatterTitleWins(t *testing.T) {
	tmp := t.TempDir()
	abs := testutil.WriteFile(t, tmp, "t.md", "---\ntitle: FM Title\n---\n\n# Heading")

	mdFile, err := LoadMarkdownFile(abs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	title, err := mdFile.GetTitle()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if title != "FM Title" {
		t.Fatalf("title = %q", title)
	}
}

// TestPlanner_extractTitleFromMDFile_LeafdocTitle 检查专用系统标题的解析与读取。
func TestPlanner_extractTitleFromMDFile_LeafdocTitle(t *testing.T) {
	tmp := t.TempDir()
	abs := testutil.WriteFile(t, tmp, "t.md", "---\nleafdoc_title: Leaf\n---\n\n# Heading")

	mdFile, err := LoadMarkdownFile(abs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	title, err := mdFile.GetTitle()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if title != "Leaf" {
		t.Fatalf("title = %q", title)
	}
}

// TestPlanner_extractTitleFromMDFile_FirstHeadingFallback 在没有元信息时使用正文一级标题。
func TestPlanner_extractTitleFromMDFile_FirstHeadingFallback(t *testing.T) {
	tmp := t.TempDir()
	abs := testutil.WriteFile(t, tmp, "t.md", "no fm\n\n# Heading Only\nx")

	mdFile, err := LoadMarkdownFile(abs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	title, err := mdFile.GetTitle()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if title != "Heading Only" {
		t.Fatalf("title = %q", title)
	}
}

// TestPlanner_extractTitleFromMDFile_FilenameFallback 确认无可用标题时去掉文件扩展名作为标题。
func TestPlanner_extractTitleFromMDFile_FilenameFallback(t *testing.T) {
	tmp := t.TempDir()
	abs := testutil.WriteFile(t, tmp, "some-file.md", "no title")

	mdFile, err := LoadMarkdownFile(abs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	title, err := mdFile.GetTitle()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if title != "some-file" {
		t.Fatalf("title = %q", title)
	}
}

// TestPlanner_extractTitleFromMDFile_FilenameFallback_WindowsPath 验证反斜线路径和文件名空格被正确处理。
func TestPlanner_extractTitleFromMDFile_FilenameFallback_WindowsPath(t *testing.T) {
	mdFile, err := NewMarkdownFileFromRaw(`C:\Users\johnjkr\AppData\Local\Temp\import-1280817455\1999-07-23 - Memo to Staff.md`, "no title")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	title, err := mdFile.GetTitle()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if title != "1999-07-23 - Memo to Staff" {
		t.Fatalf("title = %q", title)
	}
}

// TestMarkdownFile_WriteToFile_PreservesCustomFrontmatter 改变系统 ID 后写盘再解析。
// 同时核验普通字符串、列表、自有标题与正文空行，避免只检查新 ID 而遗漏其他内容。
func TestMarkdownFile_WriteToFile_PreservesCustomFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	abs := testutil.WriteFile(t, tmp, "t.md", `---
custom_key: keep-me
aliases:
  - one
leafdoc_id: old-id
leafdoc_title: Old Title
---

# Heading`)

	mdFile, err := LoadMarkdownFile(abs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	mdFile.setFrontmatterID("new-id")
	if err := mdFile.WriteToFile(); err != nil {
		t.Fatalf("WriteToFile err: %v", err)
	}

	rawBytes, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("ReadFile err: %v", err)
	}
	raw := string(rawBytes)
	if !strings.Contains(raw, "custom_key: keep-me") {
		t.Fatalf("expected custom frontmatter to be preserved, got: %q", raw)
	}
	if !strings.Contains(raw, "- one") {
		t.Fatalf("expected custom list frontmatter to be preserved, got: %q", raw)
	}

	fm, body, has, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter err: %v", err)
	}
	if !has {
		t.Fatalf("expected frontmatter after write")
	}
	if fm.LeafDocID != "new-id" {
		t.Fatalf("expected id 'new-id', got %q", fm.LeafDocID)
	}
	if fm.LeafDocTitle != "Old Title" {
		t.Fatalf("expected title 'Old Title', got %q", fm.LeafDocTitle)
	}
	if body != "\n# Heading" {
		t.Fatalf("unexpected body: %q", body)
	}
}

// TestLoadMarkdownFile_UppercaseExtension 检查大写 .MD 可通过扩展名校验并解析一级标题。
func TestLoadMarkdownFile_UppercaseExtension(t *testing.T) {
	tmp := t.TempDir()
	abs := testutil.WriteFile(t, tmp, "README.MD", "# Uppercase Extension\n\nThis file has .MD extension")

	mdFile, err := LoadMarkdownFile(abs)
	if err != nil {
		t.Fatalf("expected no error for .MD extension, got: %v", err)
	}
	title, err := mdFile.GetTitle()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if title != "Uppercase Extension" {
		t.Fatalf("title = %q, want %q", title, "Uppercase Extension")
	}
}

// TestNewMarkdownFileFromRaw_PreservesCustomFrontmatter 检查内存构造分别保存系统字段、附加字段和正文。
func TestNewMarkdownFileFromRaw_PreservesCustomFrontmatter(t *testing.T) {
	mdFile, err := NewMarkdownFileFromRaw("/tmp/test.md", `---
custom_key: keep-me
leafdoc_id: p1
leafdoc_title: Existing Title
---
# Body
Hello
`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if mdFile.GetFrontmatter().LeafDocID != "p1" {
		t.Fatalf("expected id p1, got %q", mdFile.GetFrontmatter().LeafDocID)
	}
	if mdFile.GetFrontmatter().LeafDocTitle != "Existing Title" {
		t.Fatalf("expected title 'Existing Title', got %q", mdFile.GetFrontmatter().LeafDocTitle)
	}
	if got := mdFile.GetContent(); got != "# Body\nHello\n" {
		t.Fatalf("unexpected content: %q", got)
	}
	if got := mdFile.GetFrontmatter().ExtraFields["custom_key"]; got != "keep-me" {
		t.Fatalf("expected custom_key to be preserved, got %#v", got)
	}
}

// TestNewMarkdownFileFromRaw_InvalidFrontmatter 输入不闭合 YAML 列表，必须报告解析失败。
func TestNewMarkdownFileFromRaw_InvalidFrontmatter(t *testing.T) {
	_, err := NewMarkdownFileFromRaw("/tmp/test.md", `---
leafdoc_id: [broken
---
# Body
`)
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
