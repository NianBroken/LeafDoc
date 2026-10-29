package markdown

import (
	"os"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/testutil"
)

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
