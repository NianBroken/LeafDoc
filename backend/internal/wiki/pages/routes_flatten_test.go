package pages

import (
	"testing"
)

// TestFlattenMetadataEntry_FlatString 验证普通字符串叶值按原键进入属性表。
func TestFlattenMetadataEntry_FlatString(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("key", "value", result)
	if result["key"] != "value" {
		t.Errorf("expected result[key] = value, got %q", result["key"])
	}
}

// TestFlattenMetadataEntry_NestedOneLevel 验证一层映射通过点号连接父键和子键。
func TestFlattenMetadataEntry_NestedOneLevel(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("a", map[string]interface{}{"b": "val"}, result)
	if result["a.b"] != "val" {
		t.Errorf("expected result[a.b] = val, got %q", result["a.b"])
	}
}

// TestFlattenMetadataEntry_NestedTwoLevels 验证连续映射递归形成三级属性名称。
func TestFlattenMetadataEntry_NestedTwoLevels(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("a", map[string]interface{}{
		"b": map[string]interface{}{"c": "deep"},
	}, result)
	if result["a.b.c"] != "deep" {
		t.Errorf("expected result[a.b.c] = deep, got %q", result["a.b.c"])
	}
}

// TestFlattenMetadataEntry_SkipsEmptyStringValue 检查空字符串没有产生属性键。
func TestFlattenMetadataEntry_SkipsEmptyStringValue(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("key", "", result)
	if _, ok := result["key"]; ok {
		t.Error("expected empty string to be skipped")
	}
}

// TestFlattenMetadataEntry_SkipsWhitespaceOnlyValue 验证去除空白后为空的值同样被过滤。
func TestFlattenMetadataEntry_SkipsWhitespaceOnlyValue(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("key", "   ", result)
	if _, ok := result["key"]; ok {
		t.Error("expected whitespace-only string to be skipped")
	}
}

// TestFlattenMetadataEntry_SkipsValueWithNewline 验证内部含换行的字符串不作为单行属性。
func TestFlattenMetadataEntry_SkipsValueWithNewline(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("key", "line1\nline2", result)
	if _, ok := result["key"]; ok {
		t.Error("expected multiline string to be skipped")
	}
}

// TestFlattenMetadataEntry_SkipsNonStringLeaf 使用整数验证非字符串叶值不会被自动转成文字。
func TestFlattenMetadataEntry_SkipsNonStringLeaf(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("key", 42, result)
	if _, ok := result["key"]; ok {
		t.Error("expected non-string leaf to be skipped")
	}
}

// TestFlattenMetadataEntry_SkipsLeafdocChildSegment
// 验证嵌套映射的系统前缀子键被过滤，同一映射中的普通子键仍可保留。
func TestFlattenMetadataEntry_SkipsLeafdocChildSegment(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("meta", map[string]interface{}{
		"leafdoc_id": "secret",
		"visible":    "yes",
	}, result)
	if _, ok := result["meta.leafdoc_id"]; ok {
		t.Error("expected leafdoc_ child segment to be skipped")
	}
	if result["meta.visible"] != "yes" {
		t.Errorf("expected meta.visible = yes, got %q", result["meta.visible"])
	}
}

// TestFlattenMetadataEntry_SkipsEmptyChildKey 验证空子键不会产生尾随点号属性，其他子键不受影响。
func TestFlattenMetadataEntry_SkipsEmptyChildKey(t *testing.T) {
	result := map[string]string{}
	flattenMetadataEntry("a", map[string]interface{}{
		"":  "empty-key-value",
		"b": "ok",
	}, result)
	if _, ok := result["a."]; ok {
		t.Error("expected empty child key to be skipped")
	}
	if result["a.b"] != "ok" {
		t.Errorf("expected a.b = ok, got %q", result["a.b"])
	}
}

// TestExtractPageMetadata_SkipsTagsAlways 验证标签进入独立列表，不混入普通属性。
func TestExtractPageMetadata_SkipsTagsAlways(t *testing.T) {
	fields := map[string]interface{}{
		"tags":   []interface{}{"go", "react"},
		"status": "draft",
	}
	tags, props := extractPageMetadata(fields)
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %v", tags)
	}
	if _, ok := props["tags"]; ok {
		t.Error("tags must not appear in properties")
	}
	if props["status"] != "draft" {
		t.Errorf("expected status=draft, got %q", props["status"])
	}
}

// TestExtractPageMetadata_TitleInExtraFieldsIsAlwaysCustomProperty
// 明确验证普通 title 键按自定义属性处理，不等同于 leafdoc_title 系统字段。
func TestExtractPageMetadata_TitleInExtraFieldsIsAlwaysCustomProperty(t *testing.T) {
	// 测试直接传入额外字段映射，不在此验证完整 Markdown 解析流程。
	fields := map[string]interface{}{
		"title":  "My Custom Title",
		"status": "draft",
	}
	_, props := extractPageMetadata(fields)
	if props["title"] != "My Custom Title" {
		t.Errorf("title in ExtraFields must appear in properties, got %q", props["title"])
	}
}

// TestExtractPageMetadata_SkipsLeafdocPrefixAlways 验证系统前缀键不能通过普通属性响应返回。
func TestExtractPageMetadata_SkipsLeafdocPrefixAlways(t *testing.T) {
	fields := map[string]interface{}{
		"leafdoc_id": "abc",
		"status":     "draft",
	}
	_, props := extractPageMetadata(fields)
	if _, ok := props["leafdoc_id"]; ok {
		t.Error("leafdoc_ keys must never appear in properties")
	}
}

// TestFlattenMetadataEntry_DepthLimitDoesNotPanic 验证超过配置深度的输入能够结束递归。
// 本测试只要求没有 panic，不断言被截断之后的具体属性数量。
func TestFlattenMetadataEntry_DepthLimitDoesNotPanic(t *testing.T) {
	// 构造比上限多五层的映射，让递归实际经过深度保护分支。
	inner := map[string]interface{}{"leaf": "value"}
	for i := 0; i < maxFlattenDepth+5; i++ {
		inner = map[string]interface{}{"child": inner}
	}
	result := map[string]string{}
	// 不捕获 panic，任何未正常返回的行为都会直接使测试失败。
	flattenMetadataEntry("root", inner, result)
}
