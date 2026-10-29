package revision

import (
	"bytes"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "revision") }

// Text, manifests, descriptors, cursors and pruning are all verified against MySQL.
func TestMySQLRevisionRoundTripCursorAndPruning(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	s := NewMySQLStore(dir, nil)
	body := []byte("中文历史，特殊符号 % _ ' 🌿")
	hash, err := s.SaveContentBlob("page", body)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := s.SaveAssetManifest(nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i, id := range []string{"first", "second", "third"} {
		if err = s.SaveRevision(&Revision{ID: id, PageID: "page", CreatedAt: now.Add(time.Duration(i) * time.Second), ContentHash: hash, AssetManifestHash: manifest, Title: id}); err != nil {
			t.Fatal(err)
		}
	}
	second := NewMySQLStore(dir, nil)
	items, cursor, err := second.ListRevisionsPage("page", "", 2)
	if err != nil || len(items) != 2 || items[0].ID != "third" || cursor != "second" {
		t.Fatalf("cursor page: %#v %s %v", items, cursor, err)
	}
	items, _, err = second.ListRevisionsPage("page", cursor, 2)
	if err != nil || len(items) != 1 || items[0].ID != "first" {
		t.Fatal("next page", err)
	}
	rc, err := second.OpenContentBlob("page", hash)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := io.ReadAll(rc)
	rc.Close()
	if err != nil || !bytes.Equal(actual, body) {
		t.Fatal("content mismatch", err)
	}
	if err = second.PruneRevisions("page", 1); err != nil {
		t.Fatal(err)
	}
	items, err = second.ListRevisions("page")
	if err != nil || len(items) != 1 {
		t.Fatal("prune failed", err)
	}
	if err = second.DeletePageRevisions("page"); err != nil {
		t.Fatal(err)
	}
	if _, err = second.ReadContentBlob("page", hash); !os.IsNotExist(err) {
		t.Fatalf("deleted content still present: %v", err)
	}
	if _, err = os.Stat(filepath.Join(dir, ".leafdoc", "revisions")); !os.IsNotExist(err) {
		t.Fatal("revision metadata written to disk")
	}
}
func TestRevisionAssetBlobIntegrity(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(source, []byte("resource"), 0600); err != nil {
		t.Fatal(err)
	}
	s := NewMySQLStore(dir, nil)
	hash, size, err := s.SaveAssetBlobFromPath(source)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "restored.txt")
	if err = s.CopyAssetBlobToPath(hash, size, dst); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dst)
	if err != nil || string(raw) != "resource" {
		t.Fatal(err)
	}
	if err = s.CopyAssetBlobToPath(hash, size+1, filepath.Join(dir, "bad.txt")); err == nil {
		t.Fatal("size mismatch accepted")
	}
}
func TestRevisionRejectsInvalidIDsAndUnknownRecords(t *testing.T) {
	testutil.ResetMySQL(t)
	s := NewMySQLStore(t.TempDir(), nil)
	for _, id := range []string{"", "..", "a/b", `a\b`} {
		if _, err := s.SaveContentBlob(id, []byte("body")); err == nil {
			t.Errorf("accepted %q", id)
		}
	}
	if err := s.SaveRevision(&Revision{ID: "../escape", PageID: "page", CreatedAt: time.Now()}); err == nil {
		t.Fatal("invalid revision id accepted")
	}
	if _, err := s.GetRevision("page", "missing"); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
