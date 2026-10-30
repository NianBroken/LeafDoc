package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

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

	// collect paths in a set for stable assertion (WalkDir order is OS-dependent)
	set := map[string]bool{}
	for _, e := range got {
		set[e.SourcePath] = true
		// should be slash-normalized
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

func TestImporterService_NewImporterService_RaisesZipEntryCapToMatchConfiguredAssetMax(t *testing.T) {
	planner := NewPlanner(&fakeWiki{}, tree.NewSlugService(), "")
	store := NewPlanStore()

	configuredMax := int64(200 * 1024 * 1024) // 200 MiB, larger than the 50 MiB default
	is := NewImporterService(planner, store, t.TempDir(), configuredMax)

	if got := is.zipExtractionLimits.MaxEntryBytes; got != configuredMax {
		t.Errorf("zipExtractionLimits.MaxEntryBytes = %d, want %d (configured asset max)", got, configuredMax)
	}
}

func TestImporterService_SetAssetMaxUploadSizeBytes_RaisesZipEntryCapWhenLarger(t *testing.T) {
	planner := NewPlanner(&fakeWiki{}, tree.NewSlugService(), "")
	store := NewPlanStore()
	is := NewImporterService(planner, store, t.TempDir(), 0) // defaults: 50 MiB zip entry cap

	is.SetAssetMaxUploadSizeBytes(200 * 1024 * 1024)

	if got, want := is.zipExtractionLimits.MaxEntryBytes, int64(200*1024*1024); got != want {
		t.Errorf("zipExtractionLimits.MaxEntryBytes = %d, want %d", got, want)
	}
}

func TestImporterService_SetAssetMaxUploadSizeBytes_DoesNotLowerZipEntryCap(t *testing.T) {
	planner := NewPlanner(&fakeWiki{}, tree.NewSlugService(), "")
	store := NewPlanStore()
	is := NewImporterService(planner, store, t.TempDir(), 0) // defaults: 50 MiB zip entry cap
	defaultCap := is.zipExtractionLimits.MaxEntryBytes

	is.SetAssetMaxUploadSizeBytes(10 * 1024 * 1024) // smaller than the 50 MiB default

	if got := is.zipExtractionLimits.MaxEntryBytes; got != defaultCap {
		t.Errorf("zipExtractionLimits.MaxEntryBytes = %d, want unchanged default %d", got, defaultCap)
	}
}

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
