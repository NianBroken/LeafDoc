package tree

import (
	"testing"
)

// TestGenerateUniqueChildSlug_NoConflict 验证无同级冲突时直接采用规范化名称。
func TestGenerateUniqueChildSlug_NoConflict(t *testing.T) {
	parent := &PageNode{
		Children: []*PageNode{},
	}

	s := NewSlugService()
	result := s.GenerateUniqueChildSlug(parent, "", "My Page")

	if result != "my-page" {
		t.Errorf("Expected 'my-page', got '%s'", result)
	}
}

// TestGenerateUniqueChildSlug_WithConflict 验证其他页面占用原名称时从数字后缀 1 开始避让。
func TestGenerateUniqueChildSlug_WithConflict(t *testing.T) {
	parent := &PageNode{
		Children: []*PageNode{
			{ID: "id", Slug: "my-page"},
		},
	}

	s := NewSlugService()
	result := s.GenerateUniqueChildSlug(parent, "new-id-same-parent", "My Page")

	if result != "my-page-1" {
		t.Errorf("Expected 'my-page-1', got '%s'", result)
	}
}

// TestGenerateUniqueChildSlug_MultipleConflicts 验证连续已占用名称会依次跳过，选择首个空位。
func TestGenerateUniqueChildSlug_MultipleConflicts(t *testing.T) {
	parent := &PageNode{
		Children: []*PageNode{
			{ID: "id1", Slug: "my-page"},
			{ID: "id2", Slug: "my-page-1"},
			{ID: "id3", Slug: "my-page-2"},
		},
	}

	s := NewSlugService()
	result := s.GenerateUniqueChildSlug(parent, "new-id", "My Page")

	if result != "my-page-3" {
		t.Errorf("Expected 'my-page-3', got '%s'", result)
	}
}

// TestGenerateUniqueChildSlug_SlugShouldBeTheSame 验证当前页面自身的路径段不构成冲突。
func TestGenerateUniqueChildSlug_SlugShouldBeTheSame(t *testing.T) {
	parent := &PageNode{
		Children: []*PageNode{
			{ID: "id1", Slug: "my-page"},
		},
	}

	s := NewSlugService()
	result := s.GenerateUniqueChildSlug(parent, "id1", "My Page")

	if result != "my-page" {
		t.Errorf("Expected 'my-page', got '%s'", result)
	}
}

// TestGenerateUniqueChildSlug_SpecialCharacters 验证实际字符转写结果，包括重音字母和连接符号。
// 断言以 result 的比较值为准，失败提示字符串不参与路径生成。
func TestGenerateUniqueChildSlug_SpecialCharacters(t *testing.T) {
	parent := &PageNode{}

	s := NewSlugService()
	result := s.GenerateUniqueChildSlug(parent, "", "Äpfel & Bäume!")

	if result != "apfel-and-baume" {
		t.Errorf("Expected 'aepfel-and-baume', got '%s'", result)
	}
}

// TestNormalizePath 覆盖多段名称、特殊字符、重复斜杠和前导斜杠的处理。
// 此入口按整段文字规范处理，文件扩展名中的点也转为名称连接的一部分。
func TestNormalizePath(t *testing.T) {
	s := NewSlugService()

	tests := []struct {
		input    string
		expected string
	}{
		{"folder/subfolder/page.md", "folder/subfolder/page-md"},
		{"My Folder/Another Folder/Page Title.md", "my-folder/another-folder/page-title-md"},
		{"Äpfel & Bäume/Über uns.md", "apfel-and-baume/uber-uns-md"},
		{"folder//subfolder///page.md", "folder/subfolder/page-md"},
		{"/leading/slash/page.md", "leading/slash/page-md"},
		{"only-file.md", "only-file-md"},
	}

	for _, test := range tests {

		result, err := s.NormalizePath(test.input, true)
		if err != nil {
			t.Errorf("Unexpected error for input %v: %v", test.input, err)
			continue
		}

		if result != test.expected {
			t.Errorf("For input %v, expected %v but got %v", test.input, test.expected, result)
		}
	}
}

// TestGenerateUniqueChildSlug_EmptyNormalizedInput 验证没有有效名称时直接返回空建议。
// 空文本、空白和纯标点均不得进入数字后缀循环，也不能生成以连字符开头的候选。
func TestGenerateUniqueChildSlug_EmptyNormalizedInput(t *testing.T) {
	s := NewSlugService()
	parent := &PageNode{Children: []*PageNode{{ID: "existing", Slug: "page"}}}
	for _, input := range []string{"", " \t\n", "!!!", "___", "---"} {
		t.Run(input, func(t *testing.T) {
			if got := s.GenerateUniqueChildSlug(parent, "", input); got != "" {
				t.Fatalf("GenerateUniqueChildSlug(%q) = %q, want empty", input, got)
			}
		})
	}
}

// TestIsValidSlug_AllowsUppercase 明确区分验证与生成，验证器接受大写字母而不重写。
func TestIsValidSlug_AllowsUppercase(t *testing.T) {
	s := NewSlugService()

	if err := s.IsValidSlug("ABCD-efg"); err != nil {
		t.Fatalf("expected uppercase slug to be valid, got %v", err)
	}
}

// TestGenerateValidSlug_ReservedSlugGetsSuffix 验证保留名称 api 通过后缀形成可用名称。
func TestGenerateValidSlug_ReservedSlugGetsSuffix(t *testing.T) {
	s := NewSlugService()

	if got := s.GenerateValidSlug("api"); got != "api-1" {
		t.Fatalf("GenerateValidSlug(api) = %q, want api-1", got)
	}
}

// TestNormalizePathToValidSlugs_ReservedSegmentGetsSuffix 验证多段路径逐段应用保留名称避让。
func TestNormalizePathToValidSlugs_ReservedSegmentGetsSuffix(t *testing.T) {
	s := NewSlugService()

	got, err := s.NormalizePathToValidSlugs("Reference/API")
	if err != nil {
		t.Fatalf("NormalizePathToValidSlugs err: %v", err)
	}
	if got != "reference/api-1" {
		t.Fatalf("NormalizePathToValidSlugs = %q, want reference/api-1", got)
	}
}

// TestNormalizeFilenameToValidSlug_ReservedSlugGetsSuffix 验证避让后缀位于原扩展名之前。
func TestNormalizeFilenameToValidSlug_ReservedSlugGetsSuffix(t *testing.T) {
	s := NewSlugService()

	got, err := s.NormalizeFilenameToValidSlug("API.md")
	if err != nil {
		t.Fatalf("NormalizeFilenameToValidSlug err: %v", err)
	}
	if got != "api-1.md" {
		t.Fatalf("NormalizeFilenameToValidSlug = %q, want api-1.md", got)
	}
}

// TestNormalizeFilenameToValidSlug_UnderscoreFilename 验证下划线变为连字符且保留 .md 扩展名。
func TestNormalizeFilenameToValidSlug_UnderscoreFilename(t *testing.T) {
	s := NewSlugService()

	got, err := s.NormalizeFilenameToValidSlug("CODE_OF_CONDUCT.md")
	if err != nil {
		t.Fatalf("NormalizeFilenameToValidSlug err: %v", err)
	}
	if got != "code-of-conduct.md" {
		t.Fatalf("NormalizeFilenameToValidSlug = %q, want code-of-conduct.md", got)
	}
}
