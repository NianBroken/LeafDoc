package markdown

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFM   string
		wantBody string
		wantHas  bool
	}{
		{
			name:     "no frontmatter",
			input:    "# Hello\nWorld\n",
			wantFM:   "",
			wantBody: "# Hello\nWorld\n",
			wantHas:  false,
		},
		{
			name:     "simple frontmatter",
			input:    "---\nleafdoc_id: abc123\n---\n# Title\n",
			wantFM:   "leafdoc_id: abc123",
			wantBody: "# Title\n",
			wantHas:  true,
		},
		{
			name:     "frontmatter with blank line",
			input:    "---\nleafdoc_id: abc123\n\n---\nBody\n",
			wantFM:   "leafdoc_id: abc123\n",
			wantBody: "Body\n",
			wantHas:  true,
		},
		{
			name:     "frontmatter with comments",
			input:    "---\n# comment\nleafdoc_id: abc123\n---\nBody\n",
			wantFM:   "# comment\nleafdoc_id: abc123",
			wantBody: "Body\n",
			wantHas:  true,
		},
		{
			name:     "only separator at top (no YAML)",
			input:    "---\nHello\nWorld\n---\nBody\n",
			wantFM:   "",
			wantBody: "---\nHello\nWorld\n---\nBody\n",
			wantHas:  false,
		},
		{
			name:     "horizontal rule later in document",
			input:    "# Title\n\n---\n\nText\n",
			wantFM:   "",
			wantBody: "# Title\n\n---\n\nText\n",
			wantHas:  false,
		},
		{
			name:     "unclosed frontmatter",
			input:    "---\nleafdoc_id: abc123\nBody\n",
			wantFM:   "",
			wantBody: "---\nleafdoc_id: abc123\nBody\n",
			wantHas:  false,
		},
		{
			name:     "empty frontmatter block",
			input:    "---\n---\nBody\n",
			wantFM:   "",
			wantBody: "---\n---\nBody\n",
			wantHas:  false,
		},
		{
			name:     "frontmatter with windows line endings",
			input:    "---\r\nleafdoc_id: abc123\r\n---\r\nBody\r\n",
			wantFM:   "leafdoc_id: abc123",
			wantBody: "Body\n",
			wantHas:  true,
		},
		{
			name:     "frontmatter with BOM",
			input:    "\ufeff---\nleafdoc_id: abc123\n---\nBody\n",
			wantFM:   "leafdoc_id: abc123",
			wantBody: "Body\n",
			wantHas:  true,
		},
		{
			name:     "yaml but no key colon (treated as no frontmatter)",
			input:    "---\n- item1\n- item2\n---\nBody\n",
			wantFM:   "",
			wantBody: "---\n- item1\n- item2\n---\nBody\n",
			wantHas:  false,
		},
		{
			name:     "markdown separator block with smiley is not frontmatter",
			input:    "---\n__Advertisement :)__\n---\nBody\n",
			wantFM:   "",
			wantBody: "---\n__Advertisement :)__\n---\nBody\n",
			wantHas:  false,
		},
		{
			name:     "reference-style link definition is not frontmatter",
			input:    "---\n[id]: https://example.com/demo\n---\nBody\n",
			wantFM:   "",
			wantBody: "---\n[id]: https://example.com/demo\n---\nBody\n",
			wantHas:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, has := splitFrontmatter(tt.input)

			if has != tt.wantHas {
				t.Fatalf("has = %v, want %v", has, tt.wantHas)
			}
			if fm != tt.wantFM {
				t.Fatalf("frontmatter = %q, want %q", fm, tt.wantFM)
			}
			if body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFM      Frontmatter
		wantBody    string
		wantHas     bool
		wantErr     bool
		wantErrType error
	}{
		{
			name:     "no frontmatter",
			input:    "# Hello\nWorld\n",
			wantFM:   Frontmatter{},
			wantBody: "# Hello\nWorld\n",
			wantHas:  false,
			wantErr:  false,
		},
		{
			name:  "valid frontmatter with ID only",
			input: "---\nleafdoc_id: abc123\n---\n# Title\nContent",
			wantFM: Frontmatter{
				LeafDocID: "abc123",
			},
			wantBody: "# Title\nContent",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "valid frontmatter with leafdoc_title only",
			input: "---\nleafdoc_title: My Title\n---\n# Title\nContent",
			wantFM: Frontmatter{
				LeafDocTitle: "My Title",
			},
			wantBody: "# Title\nContent",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "title as page-title alias: preserved in ExtraFields",
			input: "---\ntitle: My Title\n---\n# Title\nContent",
			wantFM: Frontmatter{
				LeafDocTitle: "My Title",
				ExtraFields: map[string]interface{}{
					"title": "My Title",
				},
			},
			wantBody: "# Title\nContent",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "both title and leafdoc_title: title preserved as custom property in ExtraFields",
			input: "---\ntitle: My Custom Title\nleafdoc_title: My Title\n---\n# Title\nContent",
			wantFM: Frontmatter{
				LeafDocTitle: "My Title",
				ExtraFields: map[string]interface{}{
					"title": "My Custom Title",
				},
			},
			wantBody: "# Title\nContent",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "valid frontmatter with both ID and title",
			input: "---\nleafdoc_id: abc123\nleafdoc_title: My Title\n---\n# Title\nContent",
			wantFM: Frontmatter{
				LeafDocID:    "abc123",
				LeafDocTitle: "My Title",
			},
			wantBody: "# Title\nContent",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "valid frontmatter with leafdoc metadata",
			input: "---\nleafdoc_id: abc123\nleafdoc_created_at: 2026-03-21T10:15:30Z\nleafdoc_updated_at: 2026-03-21T11:16:31Z\nleafdoc_creator_id: alice\nleafdoc_last_author_id: bob\n---\nBody",
			wantFM: Frontmatter{
				LeafDocID:           "abc123",
				LeafDocCreatedAt:    "2026-03-21T10:15:30Z",
				LeafDocUpdatedAt:    "2026-03-21T11:16:31Z",
				LeafDocCreatorID:    "alice",
				LeafDocLastAuthorID: "bob",
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "unknown fields are preserved",
			input: "---\nkey: value\n---\nBody",
			wantFM: Frontmatter{
				ExtraFields: map[string]interface{}{
					"key": "value",
				},
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:        "invalid YAML in frontmatter",
			input:       "---\nleafdoc_id: [invalid: yaml: structure\n---\nBody",
			wantFM:      Frontmatter{},
			wantBody:    "---\nleafdoc_id: [invalid: yaml: structure\n---\nBody",
			wantHas:     true,
			wantErr:     true,
			wantErrType: ErrFrontmatterParse,
		},
		{
			name:        "malformed YAML - unclosed brackets",
			input:       "---\nleafdoc_id: {unclosed\n---\nBody",
			wantFM:      Frontmatter{},
			wantBody:    "---\nleafdoc_id: {unclosed\n---\nBody",
			wantHas:     true,
			wantErr:     true,
			wantErrType: ErrFrontmatterParse,
		},
		{
			name:  "frontmatter with extra fields",
			input: "---\nleafdoc_id: abc123\nextra_field: ignored\n---\nBody",
			wantFM: Frontmatter{
				LeafDocID: "abc123",
				ExtraFields: map[string]interface{}{
					"extra_field": "ignored",
				},
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "template placeholder scalar is treated as string",
			input: "---\nDatum: {{date}}\n---\nBody",
			wantFM: Frontmatter{
				ExtraFields: map[string]interface{}{
					"Datum": "{{date}}",
				},
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "leafdoc_title with unquoted colon is auto-quoted and parses successfully",
			input: "---\nleafdoc_id: abc123\nleafdoc_title: ADR-0001: Filesystem as Source of Truth\n---\nBody",
			wantFM: Frontmatter{
				LeafDocID:    "abc123",
				LeafDocTitle: "ADR-0001: Filesystem as Source of Truth",
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "leafdoc_pinned bool is unaffected when leafdoc_title in the same file needs colon-quoting",
			input: "---\nleafdoc_title: ADR-0001: Filesystem as Source of Truth\nleafdoc_pinned: true\n---\nBody",
			wantFM: Frontmatter{
				LeafDocTitle:  "ADR-0001: Filesystem as Source of Truth",
				LeafDocPinned: true,
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "unquoted colon fixup and template placeholder fixup both apply in one document",
			input: "---\nleafdoc_title: ADR-0001: Filesystem as Source of Truth\nDatum: {{date}}\n---\nBody",
			wantFM: Frontmatter{
				LeafDocTitle: "ADR-0001: Filesystem as Source of Truth",
				ExtraFields: map[string]interface{}{
					"Datum": "{{date}}",
				},
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:        "unquoted colon in arbitrary ExtraFields key remains a parse error (scope guard)",
			input:       "---\nleafdoc_id: abc123\ncustom_note: Some: Value\n---\nBody",
			wantFM:      Frontmatter{},
			wantBody:    "---\nleafdoc_id: abc123\ncustom_note: Some: Value\n---\nBody",
			wantHas:     true,
			wantErr:     true,
			wantErrType: ErrFrontmatterParse,
		},
		{
			name:  "leafdoc_title with a quoted substring followed by more unquoted colon text is fixed",
			input: "---\nleafdoc_title: \"Start\" middle: end\"\n---\nBody",
			wantFM: Frontmatter{
				LeafDocTitle: "Start\" middle: end",
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "leafdoc_title starting with a backtick is fixed",
			input: "---\nleafdoc_title: `code`: an explanation\n---\nBody",
			wantFM: Frontmatter{
				LeafDocTitle: "`code`: an explanation",
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "leafdoc_title starting with @ is fixed",
			input: "---\nleafdoc_title: @mentions: how they work\n---\nBody",
			wantFM: Frontmatter{
				LeafDocTitle: "@mentions: how they work",
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "an unrelated leafdoc_title YAML comment is not corrupted when another field needs colon-fixing",
			input: "---\nleafdoc_id: something: else\nleafdoc_title: #comment: oops\n---\nBody",
			wantFM: Frontmatter{
				LeafDocID: "something: else",
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "an indented line inside an unrelated block-scalar field is not corrupted by the colon fixup",
			input: "---\nleafdoc_title: ADR-0001: Filesystem as Source of Truth\nnotes: |\n  Meeting notes.\n  leafdoc_title: v2: draft notes not real metadata\n---\nBody",
			wantFM: Frontmatter{
				LeafDocTitle: "ADR-0001: Filesystem as Source of Truth",
				ExtraFields: map[string]interface{}{
					"notes": "Meeting notes.\nleafdoc_title: v2: draft notes not real metadata",
				},
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "leafdoc_title with an unquoted template placeholder followed by more colon text is fixed",
			input: "---\nleafdoc_title: {{date}}: draft\n---\nBody",
			wantFM: Frontmatter{
				LeafDocTitle: "{{date}}: draft",
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
		{
			name:  "frontmatter with whitespace in values",
			input: "---\nleafdoc_id: \"  abc123  \"\nleafdoc_title: \"  My Title  \"\n---\nBody",
			wantFM: Frontmatter{
				LeafDocID:    "abc123",
				LeafDocTitle: "  My Title  ",
			},
			wantBody: "Body",
			wantHas:  true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, has, err := ParseFrontmatter(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.wantErrType != nil {
				if !errors.Is(err, tt.wantErrType) {
					t.Fatalf("ParseFrontmatter() error = %v, want error type %v", err, tt.wantErrType)
				}
			}

			if has != tt.wantHas {
				t.Fatalf("has = %v, want %v", has, tt.wantHas)
			}

			// WasRepaired() is asserted separately (see
			// TestParseFrontmatter_WasRepaired); this table only cares about
			// the parsed field values, so ignore it for the DeepEqual here.
			fm.repaired = false
			if !reflect.DeepEqual(fm, tt.wantFM) {
				t.Fatalf("frontmatter = %+v, want %+v", fm, tt.wantFM)
			}

			if body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestBuildMarkdownWithFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		fm      Frontmatter
		body    string
		want    string
		wantErr bool
	}{
		{
			name: "empty frontmatter struct",
			fm:   Frontmatter{},
			body: "# Title\nContent",
			want: "# Title\nContent",
		},
		{
			name: "frontmatter with empty ID",
			fm: Frontmatter{
				LeafDocID: "",
			},
			body: "# Title\nContent",
			want: "# Title\nContent",
		},
		{
			name: "frontmatter with whitespace-only ID",
			fm: Frontmatter{
				LeafDocID: "   ",
			},
			body: "# Title\nContent",
			want: "# Title\nContent",
		},
		{
			name: "frontmatter with ID only",
			fm: Frontmatter{
				LeafDocID: "abc123",
			},
			body: "# Title\nContent",
			want: "---\nleafdoc_id: abc123\n---\n# Title\nContent",
		},
		{
			name: "frontmatter with title only",
			fm: Frontmatter{
				LeafDocTitle: "My Title",
			},
			body: "# Title\nContent",
			want: "# Title\nContent",
		},
		{
			name: "frontmatter with both ID and title",
			fm: Frontmatter{
				LeafDocID:    "abc123",
				LeafDocTitle: "My Title",
			},
			body: "# Title\nContent",
			want: "---\nleafdoc_id: abc123\nleafdoc_title: My Title\n---\n# Title\nContent",
		},
		{
			name: "frontmatter with metadata fields",
			fm: Frontmatter{
				LeafDocID:           "abc123",
				LeafDocTitle:        "My Title",
				LeafDocCreatedAt:    "2026-03-21T10:15:30Z",
				LeafDocUpdatedAt:    "2026-03-21T11:16:31Z",
				LeafDocCreatorID:    "alice",
				LeafDocLastAuthorID: "bob",
			},
			body: "Content",
			want: "---\nleafdoc_id: abc123\nleafdoc_title: My Title\nleafdoc_created_at: \"2026-03-21T10:15:30Z\"\nleafdoc_updated_at: \"2026-03-21T11:16:31Z\"\nleafdoc_creator_id: alice\nleafdoc_last_author_id: bob\n---\nContent",
		},
		{
			name: "frontmatter preserves unknown fields",
			fm: Frontmatter{
				LeafDocID:    "abc123",
				LeafDocTitle: "My Title",
				ExtraFields: map[string]interface{}{
					"custom_key": "keep-me",
				},
			},
			body: "Content",
			want: "---\ncustom_key: keep-me\nleafdoc_id: abc123\nleafdoc_title: My Title\n---\nContent",
		},
		{
			name: "empty body",
			fm: Frontmatter{
				LeafDocID: "abc123",
			},
			body: "",
			want: "---\nleafdoc_id: abc123\n---\n",
		},
		{
			name: "body with newlines",
			fm: Frontmatter{
				LeafDocID: "abc123",
			},
			body: "# Title\n\nParagraph 1\n\nParagraph 2\n",
			want: "---\nleafdoc_id: abc123\n---\n# Title\n\nParagraph 1\n\nParagraph 2\n",
		},
		{
			name: "frontmatter with special characters in values",
			fm: Frontmatter{
				LeafDocID:    "abc-123_xyz",
				LeafDocTitle: "Title: With Special & Characters",
			},
			body: "Content",
			want: "---\nleafdoc_id: abc-123_xyz\nleafdoc_title: 'Title: With Special & Characters'\n---\nContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildMarkdownWithFrontmatter(tt.fm, tt.body)

			if (err != nil) != tt.wantErr {
				t.Fatalf("BuildMarkdownWithFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("BuildMarkdownWithFrontmatter() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestParseFrontmatterAndBuildRoundtrip(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantBody string
	}{
		{
			name:     "no frontmatter",
			input:    "# Title\nContent",
			wantBody: "# Title\nContent",
		},
		{
			name:     "with ID only",
			input:    "---\nleafdoc_id: abc123\n---\n# Title\nContent",
			wantBody: "# Title\nContent",
		},
		{
			name:     "with ID and title",
			input:    "---\nleafdoc_id: abc123\nleafdoc_title: My Title\n---\n# Title\nContent",
			wantBody: "# Title\nContent",
		},
		{
			name:     "with unknown fields",
			input:    "---\nleafdoc_id: abc123\ncustom_key: keep-me\n---\n# Title\nContent",
			wantBody: "# Title\nContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, has, err := ParseFrontmatter(tt.input)
			if err != nil {
				t.Fatalf("ParseFrontmatter() error = %v", err)
			}

			if body != tt.wantBody {
				t.Fatalf("body after parse = %q, want %q", body, tt.wantBody)
			}

			rebuilt, err := BuildMarkdownWithFrontmatter(fm, body)
			if err != nil {
				t.Fatalf("BuildMarkdownWithFrontmatter() error = %v", err)
			}

			fm2, body2, has2, err := ParseFrontmatter(rebuilt)
			if err != nil {
				t.Fatalf("ParseFrontmatter() second parse error = %v", err)
			}

			if has != has2 {
				t.Fatalf("has flag changed: first=%v, second=%v", has, has2)
			}

			if !reflect.DeepEqual(fm, fm2) {
				t.Fatalf("frontmatter changed: first=%+v, second=%+v", fm, fm2)
			}

			if body != body2 {
				t.Fatalf("body changed: first=%q, second=%q", body, body2)
			}
		})
	}
}

func TestParseFrontmatter_ScalarLeafDocValuesArePreserved(t *testing.T) {
	fm, body, has, err := ParseFrontmatter(`---
leafdoc_id: 123
leafdoc_title: true
---
Body`)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}
	if !has {
		t.Fatalf("expected frontmatter")
	}
	if fm.LeafDocID != "123" {
		t.Fatalf("expected numeric id to be preserved, got %q", fm.LeafDocID)
	}
	if fm.LeafDocTitle != "true" {
		t.Fatalf("expected bool title to be preserved, got %q", fm.LeafDocTitle)
	}
	if body != "Body" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestBuildMarkdownWithFrontmatter_SortsExtraFieldsDeterministically(t *testing.T) {
	fm := Frontmatter{
		LeafDocID:    "abc123",
		LeafDocTitle: "My Title",
		ExtraFields: map[string]interface{}{
			"z_key": "last",
			"a_key": "first",
		},
	}

	got, err := BuildMarkdownWithFrontmatter(fm, "Content")
	if err != nil {
		t.Fatalf("BuildMarkdownWithFrontmatter() error = %v", err)
	}

	want := `---
a_key: first
z_key: last
leafdoc_id: abc123
leafdoc_title: My Title
---
Content`
	if got != want {
		t.Fatalf("BuildMarkdownWithFrontmatter() =\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildMarkdownWithExtraFrontmatter_SortsExtraFieldsDeterministically(t *testing.T) {
	got, err := BuildMarkdownWithExtraFrontmatter(map[string]interface{}{
		"z_key": "last",
		"a_key": "first",
	}, "Content")
	if err != nil {
		t.Fatalf("BuildMarkdownWithExtraFrontmatter() error = %v", err)
	}

	want := `---
a_key: first
z_key: last
---
Content`
	if got != want {
		t.Fatalf("BuildMarkdownWithExtraFrontmatter() =\n%q\nwant:\n%q", got, want)
	}
}

func TestFrontmatter_MetadataRoundtripRFC3339(t *testing.T) {
	createdAt := time.Date(2026, time.March, 21, 10, 15, 30, 0, time.UTC).Format(time.RFC3339)
	updatedAt := time.Date(2026, time.March, 21, 11, 16, 31, 0, time.UTC).Format(time.RFC3339)

	input := Frontmatter{
		LeafDocID:           "abc123",
		LeafDocTitle:        "My Title",
		LeafDocCreatedAt:    createdAt,
		LeafDocUpdatedAt:    updatedAt,
		LeafDocCreatorID:    "alice",
		LeafDocLastAuthorID: "bob",
	}

	raw, err := BuildMarkdownWithFrontmatter(input, "Body")
	if err != nil {
		t.Fatalf("BuildMarkdownWithFrontmatter() error = %v", err)
	}

	fm, body, has, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}
	if !has {
		t.Fatalf("expected frontmatter")
	}
	if body != "Body" {
		t.Fatalf("unexpected body %q", body)
	}
	if !reflect.DeepEqual(fm, input) {
		t.Fatalf("frontmatter changed: got %+v want %+v", fm, input)
	}
}

func TestParseFrontmatter_WasRepaired(t *testing.T) {
	t.Run("clean frontmatter is not marked as repaired", func(t *testing.T) {
		fm, _, has, err := ParseFrontmatter("---\nleafdoc_id: abc123\nleafdoc_title: My Title\n---\nBody")
		if err != nil || !has {
			t.Fatalf("unexpected has=%v err=%v", has, err)
		}
		if fm.WasRepaired() {
			t.Fatalf("expected WasRepaired() to be false for clean frontmatter")
		}
	})

	t.Run("frontmatter needing the colon fixup is marked as repaired", func(t *testing.T) {
		fm, _, has, err := ParseFrontmatter("---\nleafdoc_title: ADR-0001: Filesystem as Source of Truth\n---\nBody")
		if err != nil || !has {
			t.Fatalf("unexpected has=%v err=%v", has, err)
		}
		if !fm.WasRepaired() {
			t.Fatalf("expected WasRepaired() to be true after the colon fixup ran")
		}
	})

	t.Run("a genuine parse failure is not marked as repaired", func(t *testing.T) {
		fm, _, has, err := ParseFrontmatter("---\nleafdoc_id: [invalid: yaml: structure\n---\nBody")
		if err == nil || !has {
			t.Fatalf("expected a parse error, got has=%v err=%v", has, err)
		}
		if fm.WasRepaired() {
			t.Fatalf("expected WasRepaired() to be false on a zero-value Frontmatter from a failed parse")
		}
	})
}

// TestKnownLeafDocStringFrontmatterKeys_MatchesFrontmatterStruct guards
// against knownLeafDocStringFrontmatterKeys silently drifting out of sync
// with the Frontmatter struct: every string field with a "leafdoc_*" yaml
// tag (other than the bool leafdoc_pinned) must appear in the whitelist, or
// that field silently loses unquoted-colon recovery with no compiler error.
func TestKnownLeafDocStringFrontmatterKeys_MatchesFrontmatterStruct(t *testing.T) {
	whitelisted := make(map[string]bool, len(knownLeafDocStringFrontmatterKeys))
	for _, key := range knownLeafDocStringFrontmatterKeys {
		whitelisted[key] = true
	}

	fmType := reflect.TypeOf(Frontmatter{})
	for i := 0; i < fmType.NumField(); i++ {
		field := fmType.Field(i)
		if field.Type.Kind() != reflect.String {
			continue
		}
		tag := field.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		key := strings.Split(tag, ",")[0]
		if !strings.HasPrefix(key, "leafdoc_") {
			continue
		}
		if !whitelisted[key] {
			t.Errorf("Frontmatter field %s (yaml key %q) is a leafdoc_* string field but missing from knownLeafDocStringFrontmatterKeys", field.Name, key)
		}
	}
}
