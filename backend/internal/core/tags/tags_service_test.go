// 本文件核验正文标签的列表解析、文本标准化和空值处理，不访问数据库。
package tags

import (
	"sort"
	"testing"
)

// 块列表中的标签能够被提取，辅助断言按排序后的集合比较。
func TestExtractTagsFromContent_BlockListSyntax(t *testing.T) {
	content := "---\ntags:\n  - react\n  - typescript\n---\n\n# Page"
	got := ExtractTagsFromContent(content)
	assertStringSliceEqual(t, got, []string{"react", "typescript"})
}

// 行内列表与块列表产生相同的标签集合。
func TestExtractTagsFromContent_InlineListSyntax(t *testing.T) {
	content := "---\ntags: [react, typescript]\n---\n\n# Page"
	got := ExtractTagsFromContent(content)
	assertStringSliceEqual(t, got, []string{"react", "typescript"})
}

// 不同大小写的标签值统一转为小写。
func TestExtractTagsFromContent_NormalizesToLowercase(t *testing.T) {
	content := "---\ntags:\n  - React\n  - TypeScript\n  - GO\n---\n"
	got := ExtractTagsFromContent(content)
	assertStringSliceEqual(t, got, []string{"react", "typescript", "go"})
}

// 大小写不同但标准化后相同的标签只保留一个。
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

// 引号内的边缘空白也被裁去，断言逐项检查空格与制表符。
func TestExtractTagsFromContent_TrimsWhitespace(t *testing.T) {
	content := "---\ntags:\n  - \" react \"\n  - \" go \"\n---\n"
	got := ExtractTagsFromContent(content)
	for _, tag := range got {
		if tag != trimmed(tag) {
			t.Errorf("tag %q has surrounding whitespace", tag)
		}
	}
}

// 普通正文不作为前置标签解析。
func TestExtractTagsFromContent_NoFrontmatterReturnsNil(t *testing.T) {
	content := "# Page\n\nJust content, no frontmatter."
	got := ExtractTagsFromContent(content)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// 空正文返回 nil。
func TestExtractTagsFromContent_EmptyContentReturnsNil(t *testing.T) {
	got := ExtractTagsFromContent("")
	if got != nil {
		t.Errorf("expected nil for empty content, got %v", got)
	}
}

// 存在其他前置字段但没有 tags 键时，返回 nil。
func TestExtractTagsFromContent_FrontmatterWithoutTagsFieldReturnsNil(t *testing.T) {
	content := "---\ntitle: My Page\nauthor: Alice\n---\n\n# Content"
	got := ExtractTagsFromContent(content)
	if got != nil {
		t.Errorf("expected nil when tags key absent, got %v", got)
	}
}

// 空列表按长度判定没有标签，不要求结果底层必须为 nil 切片。
func TestExtractTagsFromContent_EmptyTagsListReturnsNil(t *testing.T) {
	content := "---\ntags: []\n---\n\n# Content"
	got := ExtractTagsFromContent(content)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

// 空字符串标签被剔除，其余两个有效条目保留。
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
	// 混合大小写的 Tags 键仍作为标签字段读取。
	content := "---\nTags:\n  - react\n---\n"
	got := ExtractTagsFromContent(content)
	if len(got) != 1 || got[0] != "react" {
		t.Errorf("expected [react] for upper-case Tags key, got %v", got)
	}
}

// trimmed 为测试断言裁去两端 ASCII 空格与制表符，不代替生产代码的 Unicode 空白处理。
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

// assertStringSliceEqual 复制并排序两组标签后逐项比较，不修改调用方切片。
// 该辅助函数检查元素集合及重复数量，不检查生产函数保留的输入次序。
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
