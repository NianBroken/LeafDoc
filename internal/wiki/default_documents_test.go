package wiki

import (
	"os"
	"path/filepath"
	"testing"
	"unicode"

	"github.com/NianBroken/LeafDoc/internal/core/defaultdocs"
)

// The curriculum is complete, persisted and stable across repeated initialization.
func TestDefaultDocuments_CompleteAndIdempotent(t *testing.T) {
	w := createWikiTestInstance(t)
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	documents, err := defaultdocs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(w.tree.GetTree().Children) != 5 {
		t.Fatal("expected exactly five default pages")
	}
	for _, document := range documents {
		count := 0
		for _, r := range document.Content {
			if unicode.Is(unicode.Han, r) {
				count++
			}
		}
		if count < 2000 {
			t.Errorf("%s has only %d Chinese characters", document.Slug, count)
		}
		lookup, err := w.tree.LookupPagePath(document.Slug)
		if err != nil || !lookup.Exists {
			t.Fatalf("missing default page %s: %v", document.Slug, err)
		}
	}
	if err := w.EnsureDefaultDocuments(); err != nil {
		t.Fatal(err)
	}
	if len(w.tree.GetTree().Children) != 5 {
		t.Fatal("duplicate default pages")
	}
	// A completed marker must not repopulate a deliberately empty document tree.
	root := w.tree.GetTree()
	children := root.Children
	root.Children = nil
	if err := w.EnsureDefaultDocuments(); err != nil {
		t.Fatal(err)
	}
	if len(root.Children) != 0 {
		t.Fatal("completed initialization recreated deleted pages")
	}
	root.Children = children
}

// Pending initialization reuses its system pages and fails explicitly on invalid state.
func TestDefaultDocuments_ResumeAndRejectCorruptState(t *testing.T) {
	w := createWikiTestInstance(t)
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	state := filepath.Join(w.storageDir, defaultDocumentsStateFile)
	if err := os.WriteFile(state, []byte(`{"complete":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.EnsureDefaultDocuments(); err != nil {
		t.Fatal(err)
	}
	if len(w.tree.GetTree().Children) != 5 {
		t.Fatal("resume duplicated pages")
	}
	if err := os.WriteFile(state, []byte(`invalid`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.EnsureDefaultDocuments(); err == nil {
		t.Fatal("invalid state was accepted")
	}
}
