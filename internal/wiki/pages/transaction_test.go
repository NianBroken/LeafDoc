package pages_test

import (
	"context"
	"errors"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/search"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/wiki/pages"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
	"reflect"
	"testing"
)

type transactionFailure struct{ err error }

func (e transactionFailure) Apply(pagesave.PageSaveEvent)                    {}
func (e transactionFailure) ApplyTransactional(pagesave.PageSaveEvent) error { return e.err }

// A late failure must undo the body, version, history and search rows together.
func TestPageSaveUnitOfWorkRollsBackEveryStore(t *testing.T) {
	d := newTestDeps(t)
	index, err := search.NewMySQLIndex(d.storageDir)
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	effects := []pagesave.PageSideEffect{pagesave.NewRevisionSideEffect(d.revision, nil, nil), pagesave.NewSearchIndexSideEffect(index, d.tree, nil, nil)}
	good := pagesave.NewPageSaveOrchestrator(nil, effects...)
	created, err := pages.NewCreatePageUseCase(d.tree, d.slug, good, nil, nil).Execute(context.Background(), pages.CreatePageInput{UserID: "admin", Title: "原始内容", Slug: "transaction-original", Kind: pageKind()})
	if err != nil {
		t.Fatal(err)
	}
	before, err := database.Export(context.Background(), d.storageDir)
	if err != nil {
		t.Fatal(err)
	}
	originalVersion := created.Page.Version()
	injected := errors.New("intentional final effect failure")
	failing := pagesave.NewPageSaveOrchestrator(nil, append(effects, transactionFailure{injected})...)
	body := "改变后的中文正文与特殊符号 * []"
	_, err = pages.NewUpdatePageUseCase(d.tree, d.slug, failing, nil, nil).Execute(context.Background(), pages.UpdatePageInput{UserID: "admin", ID: created.Page.ID, Version: originalVersion, Title: "错误写入", Slug: "transaction-original", Content: &body, Kind: pageKind()})
	if !errors.Is(err, injected) {
		t.Fatalf("expected injected failure, got %v", err)
	}
	after, err := database.Export(context.Background(), d.storageDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rollback changed persisted body, history or search data")
	}
	current, err := d.tree.GetPage(created.Page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Version() != originalVersion || current.Title != created.Page.Title || current.Content != created.Page.Content {
		t.Fatal("rollback published mutable tree state")
	}
	// A subsequent request uses a fresh transaction and can persist normally.
	updated, err := pages.NewUpdatePageUseCase(d.tree, d.slug, good, nil, nil).Execute(context.Background(), pages.UpdatePageInput{UserID: "admin", ID: current.ID, Version: current.Version(), Title: "已提交", Slug: current.Slug, Content: &body, Kind: pageKind()})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Page.Content != body || updated.Page.Version() == originalVersion {
		t.Fatal("subsequent transaction did not commit")
	}
	// Failed creation must not leave a body, index, revision or visible tree node.
	before, err = database.Export(context.Background(), d.storageDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pages.NewCreatePageUseCase(d.tree, d.slug, failing, nil, nil).Execute(context.Background(), pages.CreatePageInput{UserID: "admin", Title: "不可见", Slug: "must-not-exist", Kind: pageKind()})
	if !errors.Is(err, injected) {
		t.Fatalf("expected create failure, got %v", err)
	}
	after, err = database.Export(context.Background(), d.storageDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed creation left persisted records")
	}
	if _, err = d.tree.FindPageByRoutePath("must-not-exist"); err == nil {
		t.Fatal("failed creation appeared in tree")
	}
}

// Concurrent requests never borrow another request's executor or mutable hierarchy.
func TestPageSaveConcurrentTransactionsRemainIsolated(t *testing.T) {
	d := newTestDeps(t)
	injected := errors.New("request rollback")
	good := d.orchestrator()
	bad := pagesave.NewPageSaveOrchestrator(nil, pagesave.NewRevisionSideEffect(d.revision, nil, nil), transactionFailure{injected})
	results := make(chan error, 2)
	start := make(chan struct{})
	go func() {
		<-start
		_, err := pages.NewCreatePageUseCase(d.tree, d.slug, bad, nil, nil).Execute(context.Background(), pages.CreatePageInput{UserID: "admin", Title: "回滚请求", Slug: "concurrent-rollback", Kind: pageKind()})
		results <- err
	}()
	go func() {
		<-start
		_, err := pages.NewCreatePageUseCase(d.tree, d.slug, good, nil, nil).Execute(context.Background(), pages.CreatePageInput{UserID: "admin", Title: "成功请求", Slug: "concurrent-commit", Kind: pageKind()})
		results <- err
	}()
	close(start)
	failures, successes := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if errors.Is(err, injected) {
			failures++
		} else if err == nil {
			successes++
		} else {
			t.Fatal(err)
		}
	}
	if failures != 1 || successes != 1 {
		t.Fatalf("unexpected outcomes: failures=%d successes=%d", failures, successes)
	}
	reloaded := tree.NewTreeService(d.storageDir)
	if err := reloaded.LoadTree(); err != nil {
		t.Fatal(err)
	}
	if _, err := reloaded.FindPageByRoutePath("concurrent-rollback"); err == nil {
		t.Fatal("failed request committed")
	}
	committed, err := reloaded.FindPageByRoutePath("concurrent-commit")
	if err != nil {
		t.Fatal(err)
	}
	revisions, err := d.revision.ListRevisions(committed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 1 {
		t.Fatalf("expected one committed history entry, got %d", len(revisions))
	}
}
