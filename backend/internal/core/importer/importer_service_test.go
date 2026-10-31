package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// TestFindMarkdownEntries_FindsMdRecursively_AndNormalizesSlashes 验证递归收集与后缀过滤。
// 通过路径集合检查成员，并逐个确认结果没有本机反斜杠，不在此断言排序。
func TestFindMarkdownEntries_FindsMdRecursively_AndNormalizesSlashes(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, base, "a.md", "x")
	mustWrite(t, base, "b.txt", "x")
	mustWrite(t, base, "sub/c.MD", "x")
	mustWrite(t, base, "sub/deeper/d.md", "x")

	got, err := FindMarkdownEntries(base)
	if err != nil {
		t.Fatalf("FindMarkdownEntries err: %v", err)
	}

	// 使用集合只检查收集结果，避免把这一测试与独立的排序规则绑定。
	set := map[string]bool{}
	for _, e := range got {
		set[e.SourcePath] = true
		// 对外源路径统一使用正斜杠，即使文件由本机路径函数建立。
		if strings.Contains(e.SourcePath, `\`) {
			t.Fatalf("SourcePath should be slash-normalized: %q", e.SourcePath)
		}
	}

	if !set["a.md"] {
		t.Fatalf("missing a.md, got %#v", set)
	}
	if !set["sub/c.MD"] {
		t.Fatalf("missing sub/c.MD, got %#v", set)
	}
	if !set["sub/deeper/d.md"] {
		t.Fatalf("missing sub/deeper/d.md, got %#v", set)
	}
	if set["b.txt"] {
		t.Fatalf("should not include b.txt")
	}
}

// TestFindMarkdownEntries_FindsMixedCaseMdExtensions 覆盖扩展名的三种混合大小写形式。
// 普通文本文件作为对照，确保过滤仍以 Markdown 扩展名为条件。
func TestFindMarkdownEntries_FindsMixedCaseMdExtensions(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, base, "a.MD", "x")
	mustWrite(t, base, "b.mD", "x")
	mustWrite(t, base, "c.Md", "x")
	mustWrite(t, base, "d.txt", "x")

	got, err := FindMarkdownEntries(base)
	if err != nil {
		t.Fatalf("FindMarkdownEntries err: %v", err)
	}

	set := map[string]bool{}
	for _, e := range got {
		set[e.SourcePath] = true
	}

	if !set["a.MD"] || !set["b.mD"] || !set["c.Md"] {
		t.Fatalf("expected mixed-case markdown files to be included, got %#v", set)
	}
	if set["d.txt"] {
		t.Fatalf("should not include non-markdown files")
	}
}

// TestImporterService_NewImporterService_RaisesZipEntryCapToMatchConfiguredAssetMax
// 验证构造时较大的附件上限同步成为解压单文件上限，不创建实际大文件。
func TestImporterService_NewImporterService_RaisesZipEntryCapToMatchConfiguredAssetMax(t *testing.T) {
	planner := NewPlanner(&fakeWiki{}, tree.NewSlugService(), "")
	store := NewPlanStore()

	configuredMax := int64(200 * 1024 * 1024) // 200 MiB 超过默认的 50 MiB 单文件解压限制。
	is := NewImporterService(planner, store, t.TempDir(), configuredMax)

	if got := is.zipExtractionLimits.MaxEntryBytes; got != configuredMax {
		t.Errorf("zipExtractionLimits.MaxEntryBytes = %d, want %d (configured asset max)", got, configuredMax)
	}
}

// TestImporterService_SetAssetMaxUploadSizeBytes_RaisesZipEntryCapWhenLarger
// 验证构造后的配置入口同样能提高解压单文件限制。
func TestImporterService_SetAssetMaxUploadSizeBytes_RaisesZipEntryCapWhenLarger(t *testing.T) {
	planner := NewPlanner(&fakeWiki{}, tree.NewSlugService(), "")
	store := NewPlanStore()
	is := NewImporterService(planner, store, t.TempDir(), 0) // 非正数附件参数采用默认限制，解压单文件上限为 50 MiB。

	is.SetAssetMaxUploadSizeBytes(200 * 1024 * 1024)

	if got, want := is.zipExtractionLimits.MaxEntryBytes, int64(200*1024*1024); got != want {
		t.Errorf("zipExtractionLimits.MaxEntryBytes = %d, want %d", got, want)
	}
}

// TestImporterService_SetAssetMaxUploadSizeBytes_DoesNotLowerZipEntryCap
// 验证较小附件上限不会反向降低已经设置的解压单文件上限。
func TestImporterService_SetAssetMaxUploadSizeBytes_DoesNotLowerZipEntryCap(t *testing.T) {
	planner := NewPlanner(&fakeWiki{}, tree.NewSlugService(), "")
	store := NewPlanStore()
	is := NewImporterService(planner, store, t.TempDir(), 0) // 以默认配置建立服务，再检查较小配置的影响。
	defaultCap := is.zipExtractionLimits.MaxEntryBytes

	is.SetAssetMaxUploadSizeBytes(10 * 1024 * 1024) // 10 MiB 小于已保存的 50 MiB 解压限制。

	if got := is.zipExtractionLimits.MaxEntryBytes; got != defaultCap {
		t.Errorf("zipExtractionLimits.MaxEntryBytes = %d, want unchanged default %d", got, defaultCap)
	}
}

// mustWrite 为测试创建父目录和 UTF-8 文本文件，失败直接终止所属测试并返回绝对路径。
// Helper 将失败位置归到调用方，路径按本机分隔符建立。
func mustWrite(t *testing.T, base, rel, content string) string {
	t.Helper()
	abs := filepath.Join(base, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return abs
}
