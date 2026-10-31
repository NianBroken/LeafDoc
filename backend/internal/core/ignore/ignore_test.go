// 本文件在临时目录中验证单份忽略规则、祖先合并和缓存复用。
// 案例中的模式文本属于测试输入，断言结果以实际第三方匹配器行为为准。
package ignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadIgnore 写入指定规则并通过生产读取函数加载，失败位置归于调用它的测试。
func loadIgnore(t *testing.T, tmpDir string, content string) *IgnoreFile {
	t.Helper()
	if err := os.WriteFile(filepath.Join(tmpDir, IgnoreFilename), []byte(content), 0o644); err != nil {
		t.Fatalf("write .leafdocignore: %v", err)
	}
	ig, err := LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	return ig
}

// TestLoadFromDir_MissingFile_ReturnsNil 确认不存在规则是正常状态，返回 nil 而不是空匹配器。
func TestLoadFromDir_MissingFile_ReturnsNil(t *testing.T) {
	tmp := t.TempDir()
	ig, err := LoadFromDir(tmp)
	if err != nil {
		t.Fatalf("LoadFromDir with no .leafdocignore: %v", err)
	}
	if ig != nil {
		t.Fatalf("expected nil, got %v", ig)
	}
}

// TestMatches_EmptyFile_MatchesNothing 区分空文件与文件缺失，空文件有对象但不匹配文件或目录。
func TestMatches_EmptyFile_MatchesNothing(t *testing.T) {
	tmp := t.TempDir()
	ig := loadIgnore(t, tmp, "")
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile for empty file")
	}
	if ig.Matches("any.md", false) {
		t.Fatal("expected empty file to match nothing")
	}
	if ig.Matches("some/dir", true) {
		t.Fatal("expected empty file to match nothing for dir")
	}
}

// TestMatches_SimpleFilePattern 验证后缀通配只命中对应文件，不误伤其他扩展名。
func TestMatches_SimpleFilePattern(t *testing.T) {
	tmp := t.TempDir()
	ig := loadIgnore(t, tmp, "*.log")
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("debug.log", false) {
		t.Fatal("expected *.log to match debug.log")
	}
	if ig.Matches("notes.md", false) {
		t.Fatal("expected *.log to NOT match notes.md")
	}
}

// TestMatches_DirectoryOnlyPattern 对同名路径分别传目录与文件标记，验证末尾斜线语义。
func TestMatches_DirectoryOnlyPattern(t *testing.T) {
	tmp := t.TempDir()
	ig := loadIgnore(t, tmp, "drafts/")
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("drafts", true) {
		t.Fatal("expected drafts/ to match drafts directory")
	}
	if ig.Matches("drafts", false) {
		t.Fatal("expected drafts/ to NOT match drafts file")
	}
}

// TestMatches_Negation 确认后面的感叹号规则能取消前面通配规则对指定文件的忽略。
func TestMatches_Negation(t *testing.T) {
	tmp := t.TempDir()
	ig := loadIgnore(t, tmp, "*.md\n!important.md")
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("readme.md", false) {
		t.Fatal("expected *.md to match readme.md")
	}
	if ig.Matches("important.md", false) {
		t.Fatal("expected !important.md to un-ignore important.md")
	}
}

// TestMatches_RootAnchored 比较根层与子层同名目录，验证前导斜线限制匹配位置。
func TestMatches_RootAnchored(t *testing.T) {
	tmp := t.TempDir()
	ig := loadIgnore(t, tmp, "/build/")
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("build", true) {
		t.Fatal("expected /build/ to match root-level build/")
	}
	if ig.Matches("docs/build", true) {
		t.Fatal("expected /build/ to NOT match nested build/")
	}
}

// TestMatches_Globstar 检查双星号跨越多层目录，同时保留给定路径前缀约束。
func TestMatches_Globstar(t *testing.T) {
	tmp := t.TempDir()
	ig := loadIgnore(t, tmp, "temp/**/notes.md")
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("temp/a/b/notes.md", false) {
		t.Fatal("expected temp/**/notes.md to match nested path")
	}
	if ig.Matches("notes.md", false) {
		t.Fatal("expected temp/**/notes.md to NOT match root notes.md")
	}
}

// TestMatches_CommentsAndBlanks 让注释和空行与真实规则共存，确认它们不产生额外匹配。
func TestMatches_CommentsAndBlanks(t *testing.T) {
	tmp := t.TempDir()
	ig := loadIgnore(t, tmp, "# comment\n\n*.md\n")
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("test.md", false) {
		t.Fatal("expected *.md to match test.md")
	}
	if ig.Matches("test.log", false) {
		t.Fatal("expected *.md to NOT match test.log")
	}
}

// TestPatternCount 检查 nil 对象计数为零，实际规则计数排除注释与空行。
func TestPatternCount(t *testing.T) {
	t.Run("nil returns 0", func(t *testing.T) {
		var ig *IgnoreFile
		if got := ig.PatternCount(); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("counts non-comment non-blank lines", func(t *testing.T) {
		tmp := t.TempDir()
		ig := loadIgnore(t, tmp, "# comment\n\n*.log\n*.tmp\n")
		if ig == nil {
			t.Fatal("expected non-nil IgnoreFile")
		}
		if got := ig.PatternCount(); got != 2 {
			t.Fatalf("expected 2 patterns, got %d", got)
		}
	})
}

// TestCompileLines 直接编译规则数组，核验正匹配、否定例外、未命中路径与规则数量。
func TestCompileLines(t *testing.T) {
	ig := CompileLines([]string{"*.md", "!important.md"})
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("readme.md", false) {
		t.Fatal("expected *.md to match readme.md")
	}
	if ig.Matches("important.md", false) {
		t.Fatal("expected !important.md to un-ignore important.md")
	}
	if ig.Matches("notes.txt", false) {
		t.Fatal("expected no pattern to match notes.txt")
	}
	if ig.PatternCount() != 2 {
		t.Fatalf("expected 2 patterns, got %d", ig.PatternCount())
	}
}

// TestCache_NoIgnoreFiles_ReturnsNil 检查根目录和子目录均无规则时缓存返回 nil。
func TestCache_NoIgnoreFiles_ReturnsNil(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	subdir := filepath.Join(root, "docs")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	c := NewCache(root)
	ig := c.Get(subdir)
	if ig != nil {
		t.Fatalf("expected nil, got %v", ig)
	}

	ig = c.Get(root)
	if ig != nil {
		t.Fatalf("expected nil for root, got %v", ig)
	}
}

// TestCache_RootOnly 验证子目录继承根目录规则，根目录自身读取也获得规则对象。
func TestCache_RootOnly(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	subdir := filepath.Join(root, "docs")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, IgnoreFilename), []byte("*.log"), 0o644); err != nil {
		t.Fatalf("write root .leafdocignore: %v", err)
	}

	c := NewCache(root)

	ig := c.Get(subdir)
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("debug.log", false) {
		t.Fatal("expected *.log to match debug.log")
	}
	if ig.Matches("notes.md", false) {
		t.Fatal("expected *.log to NOT match notes.md")
	}
	if ig.PatternCount() != 1 {
		t.Fatalf("expected 1 pattern, got %d", ig.PatternCount())
	}

	ig = c.Get(root)
	if ig == nil {
		t.Fatal("expected non-nil for root")
	}
}

// TestCache_MultiLevel 合并根层通配和子层否定规则，检查子层例外及未命中扩展名。
func TestCache_MultiLevel(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	subdir := filepath.Join(root, "docs")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, IgnoreFilename), []byte("*.md"), 0o644); err != nil {
		t.Fatalf("write root .leafdocignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, IgnoreFilename), []byte("!important.md"), 0o644); err != nil {
		t.Fatalf("write docs .leafdocignore: %v", err)
	}

	c := NewCache(root)

	ig := c.Get(subdir)
	if ig == nil {
		t.Fatal("expected non-nil IgnoreFile")
	}
	if !ig.Matches("docs/readme.md", false) {
		t.Fatal("expected *.md to match docs/readme.md")
	}
	if ig.Matches("docs/important.md", false) {
		t.Fatal("expected !important.md to un-ignore docs/important.md")
	}
	if ig.Matches("docs/notes.txt", false) {
		t.Fatal("expected no pattern to match docs/notes.txt")
	}
}

// TestCache_ChildOnly 验证仅有子目录规则时根目录结果为空，子目录仍能独立加载。
func TestCache_ChildOnly(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	subdir := filepath.Join(root, "docs")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(subdir, IgnoreFilename), []byte("*.log"), 0o644); err != nil {
		t.Fatalf("write docs .leafdocignore: %v", err)
	}

	c := NewCache(root)

	ig := c.Get(root)
	if ig != nil {
		t.Fatalf("expected nil for root, got %v", ig)
	}

	ig = c.Get(subdir)
	if ig == nil {
		t.Fatal("expected non-nil for subdir")
	}
	if !ig.Matches("docs/debug.log", false) {
		t.Fatal("expected *.log to match docs/debug.log")
	}
	if ig.Matches("docs/notes.md", false) {
		t.Fatal("expected *.log to NOT match docs/notes.md")
	}
}

// TestCache_Caching 连续读取同目录，检查返回对象指针相同，确认结果确实被复用。
func TestCache_Caching(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	subdir := filepath.Join(root, "docs")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, IgnoreFilename), []byte("*.log"), 0o644); err != nil {
		t.Fatalf("write root .leafdocignore: %v", err)
	}

	c := NewCache(root)

	ig1 := c.Get(subdir)
	if ig1 == nil {
		t.Fatal("expected non-nil on first call")
	}

	ig2 := c.Get(subdir)
	if ig2 == nil {
		t.Fatal("expected non-nil on second call")
	}
	if ig1 != ig2 {
		t.Fatal("expected pointer equality, got different objects")
	}
}

// TestLoadFromDir_ErrorOnDirectory 以同名目录占用规则路径，确认读取函数给出明确类型错误。
func TestLoadFromDir_ErrorOnDirectory(t *testing.T) {
	tmp := t.TempDir()
	dirPath := filepath.Join(tmp, IgnoreFilename)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_, err := LoadFromDir(tmp)
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("expected error about directory, got %v", err)
	}
}

// TestLoadFromDir_ErrorOnInvalidPattern 使用未闭合方括号文本，确认当前编译入口仍返回规则对象。
// 这个用例检查加载结果，不断言该模式能匹配哪些路径。
func TestLoadFromDir_ErrorOnInvalidPattern(t *testing.T) {
	tmp := t.TempDir()
	content := "[invalid\n"
	if err := os.WriteFile(filepath.Join(tmp, IgnoreFilename), []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	ig, err := LoadFromDir(tmp)
	if err != nil {
		t.Fatalf("LoadFromDir with potentially invalid pattern: %v", err)
	}
	if ig == nil {
		t.Fatal("expected non-nil even with questionable patterns")
	}
}
