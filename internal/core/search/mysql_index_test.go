package search

import (
	"strings"
	"testing"
)

func TestExtractHeadings_SingleH1(t *testing.T) {
	got := extractHeadings("# Hello World")
	if !strings.Contains(got, "Hello World") {
		t.Errorf("expected heading text, got %q", got)
	}
}

func TestExtractHeadings_MultipleHeadings(t *testing.T) {
	input := "# First\n## Second\n### Third"
	got := extractHeadings(input)
	for _, want := range []string{"First", "Second", "Third"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in result, got %q", want, got)
		}
	}
}

func TestExtractHeadings_InlineFormatting(t *testing.T) {
	got := extractHeadings("## **Bold** and _italic_ heading")
	if strings.Contains(got, "**") || strings.Contains(got, "_") {
		t.Errorf("expected formatting markers stripped, got %q", got)
	}
	if !strings.Contains(got, "Bold") || !strings.Contains(got, "italic") || !strings.Contains(got, "heading") {
		t.Errorf("expected heading words preserved, got %q", got)
	}
}

func TestExtractHeadings_IgnoresBodyText(t *testing.T) {
	got := extractHeadings("# Title\n\nSome body paragraph that should not appear.")
	if strings.Contains(got, "body paragraph") {
		t.Errorf("body text should not appear in headings, got %q", got)
	}
	if !strings.Contains(got, "Title") {
		t.Errorf("expected heading text, got %q", got)
	}
}

func TestExtractHeadings_NoHeadings(t *testing.T) {
	got := extractHeadings("Just plain text without any heading.")
	if got != "" {
		t.Errorf("expected empty result for no headings, got %q", got)
	}
}

func TestExtractHeadings_EmptyInput(t *testing.T) {
	got := extractHeadings("")
	if got != "" {
		t.Errorf("expected empty result for empty input, got %q", got)
	}
}

func TestExtractHeadings_HeadingWithCodeSpan(t *testing.T) {
	got := extractHeadings("## Heading `code` here")
	if strings.Contains(got, "`") {
		t.Errorf("expected backticks stripped, got %q", got)
	}
	if !strings.Contains(got, "Heading") || !strings.Contains(got, "here") {
		t.Errorf("expected heading words preserved, got %q", got)
	}
}
