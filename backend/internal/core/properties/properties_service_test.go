// 本文件直接解析前置字段，核验可检索属性的类型、系统键过滤与嵌套展开规则。
// 测试不连接数据库，断言结果只覆盖提取函数返回的内存映射。
package properties

import (
	"strconv"
	"strings"
	"testing"
)

// 普通单行字符串成为 text 属性，键和值均保留预期内容。
func TestExtractPropertiesFromContent_TextValue(t *testing.T) {
	content := "---\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// 多个平级字符串分别形成属性，检查条目数量及每个条目的值和类型。
func TestExtractPropertiesFromContent_MultipleStringValues(t *testing.T) {
	content := "---\nstatus: draft\nauthor: alice\nenvironment: staging\n---\n"
	got := ExtractPropertiesFromContent(content)

	if len(got) != 3 {
		t.Fatalf("expected 3 properties, got %d: %v", len(got), got)
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
	assertEntry(t, got, "author", PropertyEntry{Value: "alice", Type: "text"})
	assertEntry(t, got, "environment", PropertyEntry{Value: "staging", Type: "text"})
}

// YAML 整数与小数不会转换为文字，旁边的合法字符串仍进入结果。
func TestExtractPropertiesFromContent_SkipsNumberValues(t *testing.T) {
	content := "---\nscore: 42\nrating: 4.5\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["score"]; ok {
		t.Error("integer value must not be stored as a property")
	}
	if _, ok := got["rating"]; ok {
		t.Error("float value must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// true 与 false 都不参与属性索引，避免将布尔值隐式转换成字符串。
func TestExtractPropertiesFromContent_SkipsBooleanValues(t *testing.T) {
	content := "---\nfeatured: true\narchived: false\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["featured"]; ok {
		t.Error("boolean true must not be stored as a property")
	}
	if _, ok := got["archived"]; ok {
		t.Error("boolean false must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// 明确写出的空字符串被跳过，非空字段不受影响。
func TestExtractPropertiesFromContent_SkipsEmptyStringValues(t *testing.T) {
	content := "---\nempty: \"\"\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["empty"]; ok {
		t.Error("empty string value must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// 只有空白的字符串在裁去边缘空白后为空，因此不生成属性。
func TestExtractPropertiesFromContent_SkipsWhitespaceOnlyValues(t *testing.T) {
	content := "---\nblank: \"   \"\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["blank"]; ok {
		t.Error("whitespace-only string value must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// 有效字符串的首尾空格被裁去，保存实际文字。
func TestExtractPropertiesFromContent_TrimsValueWhitespace(t *testing.T) {
	content := "---\nstatus: \"  draft  \"\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// 含内部换行的块字符串不形成属性，其他单行字段仍正常提取。
func TestExtractPropertiesFromContent_SkipsMultilineStringValues(t *testing.T) {
	content := "---\ndescription: |\n  line one\n  line two\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["description"]; ok {
		t.Error("multi-line string value must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// tags 属于独立标签索引，不重复保存为属性。
func TestExtractPropertiesFromContent_SkipsTagsKey(t *testing.T) {
	content := "---\ntags:\n  - react\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["tags"]; ok {
		t.Error("'tags' must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// 系统 tags 键的过滤忽略大小写。
func TestExtractPropertiesFromContent_SkipsTagsKeyCaseInsensitive(t *testing.T) {
	content := "---\nTags:\n  - react\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["Tags"]; ok {
		t.Error("'Tags' (mixed case) must not be stored as a property")
	}
}

func TestExtractPropertiesFromContent_IndexesTitleAlone(t *testing.T) {
	// title 保存在额外字段中，作为自定义属性提取，不要求同时存在 leafdoc_title。
	content := "---\ntitle: My Page\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["title"]; !ok {
		t.Error("'title' must be indexed as a custom property")
	}
}

func TestExtractPropertiesFromContent_IndexesMixedCaseTitleAsCustomProperty(t *testing.T) {
	// Title 保留原键名并作为自定义属性，不被当作系统标题字段过滤。
	content := "---\nTitle: My Page\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["Title"]; !ok {
		t.Error("'Title' (mixed case) is a custom property and must be indexed")
	}
}

func TestExtractPropertiesFromContent_IndexesTitleWhenLeafdocTitleAlsoPresent(t *testing.T) {
	// 系统标题与自定义 title 同时出现时，自定义字段仍保留自己的值。
	content := "---\ntitle: My Custom Title\nleafdoc_title: My Title\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "title", PropertyEntry{Value: "My Custom Title", Type: "text"})
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

func TestExtractPropertiesFromContent_IndexesTitleAloneAsCustomProperty(t *testing.T) {
	// 只有 title 这一项也会生成自定义属性，不要求存在其他普通字段。
	content := "---\ntitle: My Page\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["title"]; !ok {
		t.Error("'title' must be indexed as a custom property")
	}
}

// 构造系统字段与普通字段混合的正文，确认普通 status 属性仍可提取。
// 循环按当前长度条件检查结果键，精确保留现有断言范围。
func TestExtractPropertiesFromContent_SkipsLeafdocPrefix(t *testing.T) {
	content := "---\nleafdoc_id: abc123\nleafdoc_created: 2024-01-01\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	for key := range got {
		if len(key) >= 9 && key[:9] == "leafdoc_" {
			t.Errorf("key %q with leafdoc_ prefix must not be stored", key)
		}
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// 保留前缀过滤忽略大小写，混合大小写的系统 ID 不进入索引。
func TestExtractPropertiesFromContent_SkipsLeafdocPrefixCaseInsensitive(t *testing.T) {
	content := "---\nLeafdoc_ID: abc\nstatus: ok\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["Leafdoc_ID"]; ok {
		t.Error("'Leafdoc_ID' must not be stored (reserved prefix, any case)")
	}
}

// 行内列表不展开成属性，普通字符串继续保留。
func TestExtractPropertiesFromContent_SkipsListValues(t *testing.T) {
	content := "---\nkeywords: [go, testing]\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["keywords"]; ok {
		t.Error("list values must not be stored as properties")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

// 块列表同样不参与属性提取。
func TestExtractPropertiesFromContent_SkipsBlockListValues(t *testing.T) {
	content := "---\nkeywords:\n  - go\n  - testing\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["keywords"]; ok {
		t.Error("block list values must not be stored as properties")
	}
}

// 没有值的 YAML 字段解析为空值，不保存为文字属性。
func TestExtractPropertiesFromContent_SkipsNilValues(t *testing.T) {
	content := "---\nempty:\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["empty"]; ok {
		t.Error("nil/empty values must not be stored as properties")
	}
}

// 无前置字段时返回 nil，而不是解析正文中的普通文字。
func TestExtractPropertiesFromContent_NoFrontmatterReturnsNil(t *testing.T) {
	got := ExtractPropertiesFromContent("# Page\n\nNo frontmatter.")
	if got != nil {
		t.Errorf("expected nil for content without frontmatter, got %v", got)
	}
}

// 空内容没有可提取属性，返回 nil。
func TestExtractPropertiesFromContent_EmptyContentReturnsNil(t *testing.T) {
	got := ExtractPropertiesFromContent("")
	if got != nil {
		t.Errorf("expected nil for empty content, got %v", got)
	}
}

// 所有字段均为系统键时，结果为空并统一返回 nil。
func TestExtractPropertiesFromContent_OnlyReservedKeysReturnsNil(t *testing.T) {
	content := "---\ntags:\n  - go\nleafdoc_id: abc\n---\n"
	got := ExtractPropertiesFromContent(content)
	if got != nil {
		t.Errorf("expected nil when all keys are reserved, got %v", got)
	}
}

// 没有任何字符串叶值时，不返回空映射作为有效属性集合。
func TestExtractPropertiesFromContent_OnlyNonStringValuesReturnsNil(t *testing.T) {
	content := "---\nscore: 42\nfeatured: true\nrating: 4.5\n---\n"
	got := ExtractPropertiesFromContent(content)
	if got != nil {
		t.Errorf("expected nil when all values are non-string, got %v", got)
	}
}

// 嵌套映射生成点号连接的叶键，中间映射键本身不形成属性。
func TestExtractPropertiesFromContent_NestedMapOneLevelDeep(t *testing.T) {
	content := "---\na:\n  b: value\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "a.b", PropertyEntry{Value: "value", Type: "text"})
	if _, ok := got["a"]; ok {
		t.Error("intermediate key 'a' must not appear as a property")
	}
}

// 两层子映射按实际路径连接为 a.b.c。
func TestExtractPropertiesFromContent_NestedMapTwoLevelsDeep(t *testing.T) {
	content := "---\na:\n  b:\n    c: deep\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "a.b.c", PropertyEntry{Value: "deep", Type: "text"})
}

// 同一父键下的多个字符串叶值分别形成条目。
func TestExtractPropertiesFromContent_NestedMapMultipleChildren(t *testing.T) {
	content := "---\na:\n  b: val1\n  c: val2\n---\n"
	got := ExtractPropertiesFromContent(content)
	if len(got) != 2 {
		t.Fatalf("expected 2 properties, got %d: %v", len(got), got)
	}
	assertEntry(t, got, "a.b", PropertyEntry{Value: "val1", Type: "text"})
	assertEntry(t, got, "a.c", PropertyEntry{Value: "val2", Type: "text"})
}

// 平级属性与嵌套属性可以同时进入结果，互不覆盖不同键名。
func TestExtractPropertiesFromContent_MixedFlatAndNestedKeys(t *testing.T) {
	content := "---\nstatus: draft\na:\n  b: nested\n---\n"
	got := ExtractPropertiesFromContent(content)
	if len(got) != 2 {
		t.Fatalf("expected 2 properties, got %d: %v", len(got), got)
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
	assertEntry(t, got, "a.b", PropertyEntry{Value: "nested", Type: "text"})
}

// 嵌套数字叶值被跳过，同一映射下的字符串仍保留。
func TestExtractPropertiesFromContent_NestedMapSkipsNonStringLeaves(t *testing.T) {
	content := "---\na:\n  b: 42\n  c: value\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["a.b"]; ok {
		t.Error("nested integer value must not be stored as a property")
	}
	assertEntry(t, got, "a.c", PropertyEntry{Value: "value", Type: "text"})
}

// 空串过滤同样适用于嵌套叶值。
func TestExtractPropertiesFromContent_NestedMapSkipsEmptyStringLeaves(t *testing.T) {
	content := "---\na:\n  b: \"\"\n  c: value\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["a.b"]; ok {
		t.Error("nested empty string must not be stored as a property")
	}
	assertEntry(t, got, "a.c", PropertyEntry{Value: "value", Type: "text"})
}

func TestExtractPropertiesFromContent_NestedMapSkipsLeafdocChildSegment(t *testing.T) {
	// 顶层是普通 meta 键时，子级 leafdoc_ 前缀仍被过滤，普通子字段继续提取。
	content := "---\nmeta:\n  leafdoc_id: spoofed\n  status: active\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["meta.leafdoc_id"]; ok {
		t.Error("nested leafdoc_ child segment must not be stored as a property")
	}
	assertEntry(t, got, "meta.status", PropertyEntry{Value: "active", Type: "text"})
}

func TestExtractPropertiesFromContent_NestedMapDepthLimitNotPanics(t *testing.T) {
	// 构造超过深度上限的 YAML，检查调用能够返回且输出键不超过断言中的层数范围。
	depth := maxNestedPropertyDepth + 5
	yaml := "---\n"
	indent := ""
	for i := range depth {
		yaml += indent + "k" + strconv.Itoa(i) + ":\n"
		indent += "  "
	}
	yaml += indent + "leaf: value\n---\n"

	got := ExtractPropertiesFromContent(yaml)
	for key := range got {
		if len(strings.Split(key, ".")) > maxNestedPropertyDepth+1 {
			t.Errorf("key %q exceeds max depth", key)
		}
	}
}

// assertEntry 分别报告缺失键与值类型不一致，并把失败位置归到测试调用处。
func assertEntry(t *testing.T, got map[string]PropertyEntry, key string, want PropertyEntry) {
	t.Helper()
	entry, ok := got[key]
	if !ok {
		t.Errorf("key %q missing from result %v", key, got)
		return
	}
	if entry != want {
		t.Errorf("key %q: got %+v, want %+v", key, entry, want)
	}
}
