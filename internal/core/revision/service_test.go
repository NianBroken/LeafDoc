package revision

import (
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"testing"
	"time"
)

func newRevisionTestService(t *testing.T) (*Service, *tree.TreeService, string) {
	t.Helper()
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	pages := tree.NewTreeService(dir)
	if err := pages.LoadTree(); err != nil {
		t.Fatal(err)
	}
	return NewService(dir, pages, nil), pages, dir
}
func createRevisionTestPage(t *testing.T, pages *tree.TreeService, title, slug, content string) string {
	t.Helper()
	kind := tree.NodeKindPage
	id, err := pages.CreateNode("tester", nil, title, slug, &kind)
	if err != nil {
		t.Fatal(err)
	}
	if err = pages.UpdateNode(tree.UpdateNodeInput{UserID: "tester", ID: *id, Title: title, Slug: slug, Content: &content, ExpectedVersion: tree.VersionUnchecked}); err != nil {
		t.Fatal(err)
	}
	return *id
}
func TestMySQLRevisionServiceRecordRestoreAndIntegrity(t *testing.T) {
	service, pages, dir := newRevisionTestService(t)
	id := createRevisionTestPage(t, pages, "教程", "guide", "第一版")
	first, created, err := service.RecordContentUpdate(id, "tester", "first")
	if err != nil || !created {
		t.Fatalf("record: %v", err)
	}
	if _, created, err = service.RecordContentUpdate(id, "tester", "noop"); err != nil || created {
		t.Fatalf("noop: %v", err)
	}
	body := "第二版"
	if err = pages.UpdateNode(tree.UpdateNodeInput{UserID: "tester", ID: id, Title: "教程", Slug: "guide", Content: &body, ExpectedVersion: tree.VersionUnchecked}); err != nil {
		t.Fatal(err)
	}
	if _, created, err = service.RecordContentUpdate(id, "tester", "second"); err != nil || !created {
		t.Fatal(err)
	}
	if err = service.RestoreRevision(id, first.ID, "tester"); err != nil {
		t.Fatal(err)
	}
	p, err := pages.GetPage(id)
	if err != nil || p.Content != "第一版" {
		t.Fatal("restore mismatch", err)
	}
	reloaded := tree.NewTreeService(dir)
	if err = reloaded.LoadTree(); err != nil {
		t.Fatal(err)
	}
	p, err = reloaded.GetPage(id)
	if err != nil || p.Content != "第一版" {
		t.Fatal("restore not persisted", err)
	}
	if err = database.DeleteRecord(dir, "revision_content", id+":"+first.ContentHash); err != nil {
		t.Fatal(err)
	}
	issues, err := service.CheckRevisionIntegrity(id)
	if err != nil || len(issues) == 0 {
		t.Fatal("missing database blob not detected", err)
	}
}
func TestMySQLRevisionCoalescingPreservesLatestBody(t *testing.T) {
	_, pages, dir := newRevisionTestService(t)
	service := NewService(dir, pages, nil, ServiceOptions{CoalesceWindow: time.Hour})
	id := createRevisionTestPage(t, pages, "Guide", "guide", "one")
	first, _, err := service.RecordContentUpdate(id, "tester", "first")
	if err != nil {
		t.Fatal(err)
	}
	body := "two"
	if err = pages.UpdateNode(tree.UpdateNodeInput{UserID: "tester", ID: id, Title: "Guide", Slug: "guide", Content: &body, ExpectedVersion: tree.VersionUnchecked}); err != nil {
		t.Fatal(err)
	}
	second, _, err := service.RecordContentUpdate(id, "tester", "second")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatal("same-author update did not coalesce")
	}
	items, err := service.ListRevisions(id)
	if err != nil || len(items) != 1 {
		t.Fatal("unexpected revision count", err)
	}
	raw, err := service.store.ReadContentBlob(id, second.ContentHash)
	if err != nil || string(raw) != "two" {
		t.Fatal("latest content lost", err)
	}
}
