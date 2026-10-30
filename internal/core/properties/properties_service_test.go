package properties

import (
	"strconv"
	"strings"
	"testing"
)

func TestExtractPropertiesFromContent_TextValue(t *testing.T) {
	content := "---\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

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

func TestExtractPropertiesFromContent_SkipsEmptyStringValues(t *testing.T) {
	content := "---\nempty: \"\"\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["empty"]; ok {
		t.Error("empty string value must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

func TestExtractPropertiesFromContent_SkipsWhitespaceOnlyValues(t *testing.T) {
	content := "---\nblank: \"   \"\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["blank"]; ok {
		t.Error("whitespace-only string value must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

func TestExtractPropertiesFromContent_TrimsValueWhitespace(t *testing.T) {
	content := "---\nstatus: \"  draft  \"\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

func TestExtractPropertiesFromContent_SkipsMultilineStringValues(t *testing.T) {
	content := "---\ndescription: |\n  line one\n  line two\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["description"]; ok {
		t.Error("multi-line string value must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

func TestExtractPropertiesFromContent_SkipsTagsKey(t *testing.T) {
	content := "---\ntags:\n  - react\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["tags"]; ok {
		t.Error("'tags' must not be stored as a property")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

func TestExtractPropertiesFromContent_SkipsTagsKeyCaseInsensitive(t *testing.T) {
	content := "---\nTags:\n  - react\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["Tags"]; ok {
		t.Error("'Tags' (mixed case) must not be stored as a property")
	}
}

func TestExtractPropertiesFromContent_IndexesTitleAlone(t *testing.T) {
	// "title" is always a custom property — it lands in ExtraFields and is indexed
	// regardless of whether leafdoc_title is also present.
	content := "---\ntitle: My Page\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["title"]; !ok {
		t.Error("'title' must be indexed as a custom property")
	}
}

func TestExtractPropertiesFromContent_IndexesMixedCaseTitleAsCustomProperty(t *testing.T) {
	// YAML is case-sensitive: "Title" ≠ "title". Both are custom properties.
	content := "---\nTitle: My Page\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["Title"]; !ok {
		t.Error("'Title' (mixed case) is a custom property and must be indexed")
	}
}

func TestExtractPropertiesFromContent_IndexesTitleWhenLeafdocTitleAlsoPresent(t *testing.T) {
	// Both "title" and "leafdoc_title" present: "title" is a custom property.
	content := "---\ntitle: My Custom Title\nleafdoc_title: My Title\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "title", PropertyEntry{Value: "My Custom Title", Type: "text"})
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

func TestExtractPropertiesFromContent_IndexesTitleAloneAsCustomProperty(t *testing.T) {
	// "title" alone is always a custom property — no leafdoc_title required.
	content := "---\ntitle: My Page\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["title"]; !ok {
		t.Error("'title' must be indexed as a custom property")
	}
}

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

func TestExtractPropertiesFromContent_SkipsLeafdocPrefixCaseInsensitive(t *testing.T) {
	content := "---\nLeafdoc_ID: abc\nstatus: ok\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["Leafdoc_ID"]; ok {
		t.Error("'Leafdoc_ID' must not be stored (reserved prefix, any case)")
	}
}

func TestExtractPropertiesFromContent_SkipsListValues(t *testing.T) {
	content := "---\nkeywords: [go, testing]\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["keywords"]; ok {
		t.Error("list values must not be stored as properties")
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
}

func TestExtractPropertiesFromContent_SkipsBlockListValues(t *testing.T) {
	content := "---\nkeywords:\n  - go\n  - testing\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["keywords"]; ok {
		t.Error("block list values must not be stored as properties")
	}
}

func TestExtractPropertiesFromContent_SkipsNilValues(t *testing.T) {
	content := "---\nempty:\nstatus: draft\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["empty"]; ok {
		t.Error("nil/empty values must not be stored as properties")
	}
}

func TestExtractPropertiesFromContent_NoFrontmatterReturnsNil(t *testing.T) {
	got := ExtractPropertiesFromContent("# Page\n\nNo frontmatter.")
	if got != nil {
		t.Errorf("expected nil for content without frontmatter, got %v", got)
	}
}

func TestExtractPropertiesFromContent_EmptyContentReturnsNil(t *testing.T) {
	got := ExtractPropertiesFromContent("")
	if got != nil {
		t.Errorf("expected nil for empty content, got %v", got)
	}
}

func TestExtractPropertiesFromContent_OnlyReservedKeysReturnsNil(t *testing.T) {
	content := "---\ntags:\n  - go\nleafdoc_id: abc\n---\n"
	got := ExtractPropertiesFromContent(content)
	if got != nil {
		t.Errorf("expected nil when all keys are reserved, got %v", got)
	}
}

func TestExtractPropertiesFromContent_OnlyNonStringValuesReturnsNil(t *testing.T) {
	content := "---\nscore: 42\nfeatured: true\nrating: 4.5\n---\n"
	got := ExtractPropertiesFromContent(content)
	if got != nil {
		t.Errorf("expected nil when all values are non-string, got %v", got)
	}
}

func TestExtractPropertiesFromContent_NestedMapOneLevelDeep(t *testing.T) {
	content := "---\na:\n  b: value\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "a.b", PropertyEntry{Value: "value", Type: "text"})
	if _, ok := got["a"]; ok {
		t.Error("intermediate key 'a' must not appear as a property")
	}
}

func TestExtractPropertiesFromContent_NestedMapTwoLevelsDeep(t *testing.T) {
	content := "---\na:\n  b:\n    c: deep\n---\n"
	got := ExtractPropertiesFromContent(content)
	assertEntry(t, got, "a.b.c", PropertyEntry{Value: "deep", Type: "text"})
}

func TestExtractPropertiesFromContent_NestedMapMultipleChildren(t *testing.T) {
	content := "---\na:\n  b: val1\n  c: val2\n---\n"
	got := ExtractPropertiesFromContent(content)
	if len(got) != 2 {
		t.Fatalf("expected 2 properties, got %d: %v", len(got), got)
	}
	assertEntry(t, got, "a.b", PropertyEntry{Value: "val1", Type: "text"})
	assertEntry(t, got, "a.c", PropertyEntry{Value: "val2", Type: "text"})
}

func TestExtractPropertiesFromContent_MixedFlatAndNestedKeys(t *testing.T) {
	content := "---\nstatus: draft\na:\n  b: nested\n---\n"
	got := ExtractPropertiesFromContent(content)
	if len(got) != 2 {
		t.Fatalf("expected 2 properties, got %d: %v", len(got), got)
	}
	assertEntry(t, got, "status", PropertyEntry{Value: "draft", Type: "text"})
	assertEntry(t, got, "a.b", PropertyEntry{Value: "nested", Type: "text"})
}

func TestExtractPropertiesFromContent_NestedMapSkipsNonStringLeaves(t *testing.T) {
	content := "---\na:\n  b: 42\n  c: value\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["a.b"]; ok {
		t.Error("nested integer value must not be stored as a property")
	}
	assertEntry(t, got, "a.c", PropertyEntry{Value: "value", Type: "text"})
}

func TestExtractPropertiesFromContent_NestedMapSkipsEmptyStringLeaves(t *testing.T) {
	content := "---\na:\n  b: \"\"\n  c: value\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["a.b"]; ok {
		t.Error("nested empty string must not be stored as a property")
	}
	assertEntry(t, got, "a.c", PropertyEntry{Value: "value", Type: "text"})
}

func TestExtractPropertiesFromContent_NestedMapSkipsLeafdocChildSegment(t *testing.T) {
	// A nested child key that starts with leafdoc_ must not be indexed even
	// when the top-level key is a normal user key.
	content := "---\nmeta:\n  leafdoc_id: spoofed\n  status: active\n---\n"
	got := ExtractPropertiesFromContent(content)
	if _, ok := got["meta.leafdoc_id"]; ok {
		t.Error("nested leafdoc_ child segment must not be stored as a property")
	}
	assertEntry(t, got, "meta.status", PropertyEntry{Value: "active", Type: "text"})
}

func TestExtractPropertiesFromContent_NestedMapDepthLimitNotPanics(t *testing.T) {
	// Build YAML that is deeper than maxNestedPropertyDepth. extractFlatEntry
	// must return without crashing or producing an unboundedly long key.
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
