package tree

import (
	"errors"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "tree") }
func newLoadedService(t *testing.T) (*TreeService, string) {
	t.Helper()
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	s := NewTreeService(dir)
	if err := s.LoadTree(); err != nil {
		t.Fatal(err)
	}
	return s, dir
}
func ptrKind(kind NodeKind) *NodeKind { return &kind }
func createTestNode(t *testing.T, s *TreeService, parent *string, title, slug string, kind NodeKind) string {
	t.Helper()
	id, err := s.CreateNode("tester", parent, title, slug, &kind)
	if err != nil {
		t.Fatal(err)
	}
	return *id
}

// Database reconstruction must retain hierarchy and content without any Markdown files.
func TestMySQLTreePersistenceAndVersionConflict(t *testing.T) {
	s, dir := newLoadedService(t)
	section := createTestNode(t, s, nil, "文档", "docs", NodeKindSection)
	id := createTestNode(t, s, &section, "中文页面", "guide", NodeKindPage)
	page, err := s.GetPage(id)
	if err != nil {
		t.Fatal(err)
	}
	version := page.Version()
	body := "中文内容，特殊字符 ' % _ 与 emoji 🌿"
	if err = s.UpdateNode(UpdateNodeInput{UserID: "tester", ID: id, Title: "中文页面", Slug: "guide", Content: &body, ExpectedVersion: version, Tags: []string{"教程"}, Properties: map[string]string{"状态": "完成"}}); err != nil {
		t.Fatal(err)
	}
	if err = s.UpdateNode(UpdateNodeInput{ID: id, Title: "旧版本", Slug: "guide", Content: &body, ExpectedVersion: version}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale write: %v", err)
	}
	second := NewTreeService(dir)
	if err = second.LoadTree(); err != nil {
		t.Fatal(err)
	}
	restored, err := second.GetPage(id)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Content != body || restored.Parent.ID != section {
		t.Fatalf("restored %#v", restored)
	}
	if _, err = os.Stat(filepath.Join(dir, "root")); !os.IsNotExist(err) {
		t.Fatal("page persistence created a filesystem tree")
	}
}
func TestMySQLTreeMoveRenameSortAndDelete(t *testing.T) {
	s, dir := newLoadedService(t)
	a := createTestNode(t, s, nil, "A", "a", NodeKindSection)
	b := createTestNode(t, s, nil, "B", "b", NodeKindSection)
	id := createTestNode(t, s, &a, "Guide", "guide", NodeKindPage)
	p, _ := s.GetPage(id)
	if err := s.MoveNode("tester", id, b, p.Version()); err != nil {
		t.Fatal(err)
	}
	p, _ = s.GetPage(id)
	if err := s.UpdateNode(UpdateNodeInput{UserID: "tester", ID: id, Title: "Renamed", Slug: "renamed", ExpectedVersion: p.Version()}); err != nil {
		t.Fatal(err)
	}
	if err := s.SortPages("root", []string{b, a}); err != nil {
		t.Fatal(err)
	}
	second := NewTreeService(dir)
	if err := second.LoadTree(); err != nil {
		t.Fatal(err)
	}
	if second.GetTree().Children[0].ID != b {
		t.Fatal("order not persisted")
	}
	p, _ = second.GetPage(id)
	if p.Parent.ID != b || p.Slug != "renamed" {
		t.Fatal("move/rename not persisted")
	}
	parent, _ := second.GetPage(b)
	if err := second.DeleteNode("tester", b, true, parent.Version()); err != nil {
		t.Fatal(err)
	}
	if _, err := second.GetPage(id); !errors.Is(err, ErrPageNotFound) {
		t.Fatalf("descendant remains: %v", err)
	}
}
func TestMySQLTreeRejectsCrossInstanceStaleVersion(t *testing.T) {
	a, dir := newLoadedService(t)
	id := createTestNode(t, a, nil, "A", "a", NodeKindPage)
	b := NewTreeService(dir)
	if err := b.LoadTree(); err != nil {
		t.Fatal(err)
	}
	p, _ := a.GetPage(id)
	body := "first"
	if err := a.UpdateNode(UpdateNodeInput{ID: id, Title: "A", Slug: "a", Content: &body, ExpectedVersion: p.Version()}); err != nil {
		t.Fatal(err)
	}
	old, _ := b.GetPage(id)
	body = "stale"
	if err := b.UpdateNode(UpdateNodeInput{ID: id, Title: "A", Slug: "a", Content: &body, ExpectedVersion: old.Version()}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected database version conflict: %v", err)
	}
}

// A parent-only move cannot silently become a reorder; explicit positions can.
func TestMySQLTreeSameParentMoveRequiresPosition(t *testing.T) {
	s, _ := newLoadedService(t)
	id := createTestNode(t, s, nil, "A", "a", NodeKindPage)
	p, err := s.GetPage(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MoveNode("tester", id, "root", p.Version()); !errors.Is(err, ErrInvalidOperation) {
		t.Fatalf("parent-only move was accepted: %v", err)
	}
	if err := s.MoveNodeToPosition("tester", id, "root", p.Version(), 0); err != nil {
		t.Fatal(err)
	}
}
