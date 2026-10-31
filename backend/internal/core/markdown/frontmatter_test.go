package markdown

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestSplitFrontmatter 对比前置区段、正文和识别标记，覆盖有效头部与普通 Markdown。
// 表内输入包含 BOM、Windows 换行、缺少结束线、空区段与引用式链接定义。
// 仅出现分隔线或冒号不等于有效头部，未识别时正文应保留对应文本。
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
			// 逐项比较三个结果，避免只检查头部内容而忽略正文被误删的情况。
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

// TestParseFrontmatter 检查已知字段、未知字段保留及正文拆分的完整结果。
// 标题别名仍保存在扩展字段，专用标题字段优先；标识去除边缘空白而标题保留空白。
// 兼容处理只针对实现支持的字段和占位形式，任意自定义字段中的非法冒号仍应报错。
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
			// 识别到头部后解析失败仍报告 has 为真，并将完整原文作为正文返回。
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
			// 未加引号的模板占位值按字符串保留，不执行模板展开。
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
			// 块文本中的同名文字属于 notes 内容，不应被当成顶层 LeafDoc 字段处理。
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

			// 本表核对解析字段，兼容处理标记在专门用例中断言，因此比较前归零该内部字段。
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

// TestBuildMarkdownWithFrontmatter 以完整字符串核对头部生成、字段顺序、引号和正文换行。
// 缺少有效标识时直接输出正文，即使存在标题也不据此创建 LeafDoc 头部。
// 有效标识存在时，元数据与扩展属性按生成器规定顺序输出。
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
			// 精确比较输出，字段语义相同但排版或换行不同也不能通过。
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

// TestParseFrontmatterAndBuildRoundtrip 按解析、生成、再次解析的顺序验证语义保持。
// 比较识别标记、字段结构和正文，不要求生成后的 YAML 与原始排版逐字一致。
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

// TestParseFrontmatter_ScalarLeafDocValuesArePreserved 确认数字标识和布尔标题转为字符串。
// YAML 的原始标量类型不会导致这两个字符串字段丢失。
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

// TestBuildMarkdownWithFrontmatter_SortsExtraFieldsDeterministically 将逆序扩展键写入映射。
// 输出必须按键排序扩展属性，再按既定顺序写 LeafDoc 字段，不受映射遍历顺序影响。
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

// TestBuildMarkdownWithExtraFrontmatter_SortsExtraFieldsDeterministically 验证纯扩展头部的排序规则。
// 此入口不要求 LeafDoc 标识，仍可为正文生成仅含自定义属性的头部。
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

// TestFrontmatter_MetadataRoundtripRFC3339 使用 UTC 秒级时间文本验证元数据往返。
// 时间值在这里是字符串，测试同时核对创建者、作者、标题、标识与正文。
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

// TestParseFrontmatter_WasRepaired 区分直接成功、兼容处理后成功和实际解析失败的标记。
// 普通头部与失败返回的零值不带标记，未加引号的受支持冒号值处理成功后带标记。
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

// TestKnownLeafDocStringFrontmatterKeys_MatchesFrontmatterStruct 通过反射检查兼容处理键集合。
// 所有带 leafdoc_ YAML 名称的字符串字段必须在集合内，布尔字段和非持久字段跳过。
// 此断言检查结构体到集合的覆盖，不禁止集合中存在其他键。
func TestKnownLeafDocStringFrontmatterKeys_MatchesFrontmatterStruct(t *testing.T) {
	whitelisted := make(map[string]bool, len(knownLeafDocStringFrontmatterKeys))
	for _, key := range knownLeafDocStringFrontmatterKeys {
		whitelisted[key] = true
	}

	// 只考察实际字符串字段，再提取 YAML 标签逗号之前的字段名称。
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
