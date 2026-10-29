package tree

import (
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"testing"
)

// Corrupt hierarchy is rejected instead of silently dropping or reparenting documents.
func TestMySQLReconstructionRejectsInvalidParent(t *testing.T) {
	s, _ := newLoadedService(t)
	n := &PageNode{ID: "orphan", Title: "Orphan", Slug: "orphan", Kind: NodeKindPage, Parent: &PageNode{ID: "missing"}}
	if err := s.store.writeRecord(n, "body"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.store.ReloadFromDatabase(); err == nil {
		t.Fatal("orphan was accepted")
	}
	if err := database.DeleteRecord(s.storageDir, nodeNamespace, n.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.LoadTree(); err != nil {
		t.Fatal(err)
	}
}
