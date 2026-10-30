package tags

import (
	"sort"
	"testing"
)

func TestExtractTagsFromContent_BlockListSyntax(t *testing.T) {
	content := "---\ntags:\n  - react\n  - typescript\n---\n\n# Page"
	got := ExtractTagsFromContent(content)
	assertStringSliceEqual(t, got, []string{"react", "typescript"})
}

func TestExtractTagsFromContent_InlineListSyntax(t *testing.T) {
	content := "---\ntags: [react, typescript]\n---\n\n# Page"
	got := ExtractTagsFromContent(content)
	assertStringSliceEqual(t, got, []string{"react", "typescript"})
}

func TestExtractTagsFromContent_NormalizesToLowercase(t *testing.T) {
	content := "---\ntags:\n  - React\n  - TypeScript\n  - GO\n---\n"
	got := ExtractTagsFromContent(content)
	assertStringSliceEqual(t, got, []string{"react", "typescript", "go"})
}

func TestExtractTagsFromContent_DeduplicatesCaseInsensitive(t *testing.T) {
	content := "---\ntags:\n  - react\n  - React\n  - REACT\n---\n"
	got := ExtractTagsFromContent(content)
	if len(got) != 1 {
		t.Errorf("expected 1 deduplicated tag, got %d: %v", len(got), got)
	}
	if len(got) > 0 && got[0] != "react" {
		t.Errorf("got[0] = %q, want 'react'", got[0])
	}
}

func TestExtractTagsFromContent_TrimsWhitespace(t *testing.T) {
	content := "---\ntags:\n  - \" react \"\n  - \" go \"\n---\n"
	got := ExtractTagsFromContent(content)
	for _, tag := range got {
		if tag != trimmed(tag) {
			t.Errorf("tag %q has surrounding whitespace", tag)
		}
	}
}

func TestExtractTagsFromContent_NoFrontmatterReturnsNil(t *testing.T) {
	content := "# Page\n\nJust content, no frontmatter."
	got := ExtractTagsFromContent(content)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestExtractTagsFromContent_EmptyContentReturnsNil(t *testing.T) {
	got := ExtractTagsFromContent("")
	if got != nil {
		t.Errorf("expected nil for empty content, got %v", got)
	}
}

func TestExtractTagsFromContent_FrontmatterWithoutTagsFieldReturnsNil(t *testing.T) {
	content := "---\ntitle: My Page\nauthor: Alice\n---\n\n# Content"
	got := ExtractTagsFromContent(content)
	if got != nil {
		t.Errorf("expected nil when tags key absent, got %v", got)
	}
}

func TestExtractTagsFromContent_EmptyTagsListReturnsNil(t *testing.T) {
	content := "---\ntags: []\n---\n\n# Content"
	got := ExtractTagsFromContent(content)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestExtractTagsFromContent_SkipsEmptyTagEntries(t *testing.T) {
	content := "---\ntags:\n  - react\n  - \"\"\n  - typescript\n---\n"
	got := ExtractTagsFromContent(content)
	for _, tag := range got {
		if tag == "" {
			t.Errorf("empty tag should be filtered out")
		}
	}
	if len(got) != 2 {
		t.Errorf("expected 2 non-empty tags, got %d: %v", len(got), got)
	}
}

func TestExtractTagsFromContent_TagsKeyIsCaseInsensitive(t *testing.T) {
	// Frontmatter keys like "Tags" or "TAGS" should still be found.
	content := "---\nTags:\n  - react\n---\n"
	got := ExtractTagsFromContent(content)
	if len(got) != 1 || got[0] != "react" {
		t.Errorf("expected [react] for upper-case Tags key, got %v", got)
	}
}

func trimmed(s string) string {
	result := s
	for len(result) > 0 && (result[0] == ' ' || result[0] == '\t') {
		result = result[1:]
	}
	for len(result) > 0 && (result[len(result)-1] == ' ' || result[len(result)-1] == '\t') {
		result = result[:len(result)-1]
	}
	return result
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()

	gc := append([]string(nil), got...)
	wc := append([]string(nil), want...)
	sort.Strings(gc)
	sort.Strings(wc)

	if len(gc) != len(wc) {
		t.Errorf("len = %d, want %d\n got:  %v\n want: %v", len(gc), len(wc), gc, wc)
		return
	}
	for i := range gc {
		if gc[i] != wc[i] {
			t.Errorf("[%d] = %q, want %q\n got:  %v\n want: %v", i, gc[i], wc[i], gc, wc)
			return
		}
	}
}
