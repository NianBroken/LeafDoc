package links

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/testutil"
)

func TestLinksStore_CreatesNoLocalDatabase(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	store, err := NewLinksStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected local persistence: %v", entries)
	}
}

func TestLinksStore_UsesInnoDB(t *testing.T) {
	testutil.ResetMySQL(t)
	store, err := NewLinksStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var engine string
	if err := store.db.QueryRow("SELECT ENGINE FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", "links").Scan(&engine); err != nil {
		t.Fatal(err)
	}
	if engine != "InnoDB" {
		t.Fatalf("engine=%s", engine)
	}
}

// Pins the RWMutex change (mu sync.Mutex -> sync.RWMutex, read methods
// moved to RLock): concurrent read calls must be able to run in parallel
// without data races or errors. Run with `go test -race` to actually catch
// a regression (e.g. a read method left on Lock by mistake would still
// pass this test functionally, but a write method accidentally moved to
// RLock would show up as a race here against Close/AddLinks-style
// mutation, which -race is what would catch).
func TestLinksStore_ConcurrentReadsDoNotRace(t *testing.T) {
	testutil.ResetMySQL(t)
	tmp := t.TempDir()
	store, err := NewLinksStore(tmp)
	if err != nil {
		t.Fatalf("NewLinksStore err: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)

	const pageCount = 20
	pageIDs := make([]string, pageCount)
	for i := 0; i < pageCount; i++ {
		pageID := fmt.Sprintf("page-%d", i)
		pageIDs[i] = pageID
		if err := store.AddLinks(pageID, "Title "+pageID, []TargetLink{{
			TargetPageID:   "target-" + pageID,
			TargetPagePath: "target/" + pageID,
		}}); err != nil {
			t.Fatalf("AddLinks(%s) failed: %v", pageID, err)
		}
	}

	var wg sync.WaitGroup
	errCh := make(chan error, pageCount*3)
	for _, pageID := range pageIDs {
		pageID := pageID
		wg.Add(3)
		go func() {
			defer wg.Done()
			if _, err := store.GetBacklinksForPage(pageID); err != nil {
				errCh <- err
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := store.GetOutgoingLinksForPage(pageID); err != nil {
				errCh <- err
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := store.GetOutgoingLinksForPages(pageIDs); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent read failed: %v", err)
	}
}

func TestLinksStore_GetOutgoingLinksForPages_BatchesLargeInputs(t *testing.T) {
	testutil.ResetMySQL(t)
	tmp := t.TempDir()
	store, err := NewLinksStore(tmp)
	if err != nil {
		t.Fatalf("NewLinksStore err: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)

	pageIDs := make([]string, 0, maxOutgoingLinksQueryArgs+5)
	for i := 0; i < maxOutgoingLinksQueryArgs+5; i++ {
		pageID := fmt.Sprintf("page-%d", i)
		pageIDs = append(pageIDs, pageID)
		if err := store.AddLinks(pageID, "Title "+pageID, []TargetLink{{
			TargetPageID:   "target-" + pageID,
			TargetPagePath: "target/" + pageID,
		}}); err != nil {
			t.Fatalf("AddLinks(%s) failed: %v", pageID, err)
		}
	}

	outgoingByPageID, err := store.GetOutgoingLinksForPages(pageIDs)
	if err != nil {
		t.Fatalf("GetOutgoingLinksForPages failed: %v", err)
	}
	if len(outgoingByPageID) != len(pageIDs) {
		t.Fatalf("expected %d page entries, got %d", len(pageIDs), len(outgoingByPageID))
	}

	for _, pageID := range pageIDs {
		outgoings := outgoingByPageID[pageID]
		if len(outgoings) != 1 {
			t.Fatalf("expected 1 outgoing for %s, got %d", pageID, len(outgoings))
		}
		if outgoings[0].FromPageID != pageID {
			t.Fatalf("expected outgoing from %s, got %s", pageID, outgoings[0].FromPageID)
		}
		if outgoings[0].ToPath != "target/"+pageID {
			t.Fatalf("expected target path %q, got %q", "target/"+pageID, outgoings[0].ToPath)
		}
	}
}

func TestLinksStore_GetBrokenLinks_ReturnsOnlyBrokenOrderedByToPathThenFromTitle(t *testing.T) {
	testutil.ResetMySQL(t)
	tmp := t.TempDir()
	store, err := NewLinksStore(tmp)
	if err != nil {
		t.Fatalf("NewLinksStore err: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)

	if err := store.AddLinks("p1", "Alpha", []TargetLink{
		{TargetPagePath: "missing/zeta", State: LinkBroken},
		{TargetPagePath: "exists/ok", State: LinkResolved},
	}); err != nil {
		t.Fatalf("AddLinks(p1) failed: %v", err)
	}
	if err := store.AddLinks("p2", "Beta", []TargetLink{
		{TargetPagePath: "missing/alpha", State: LinkBroken},
	}); err != nil {
		t.Fatalf("AddLinks(p2) failed: %v", err)
	}
	if err := store.AddLinks("p3", "Gamma", []TargetLink{
		{TargetPagePath: "wikilink:Ghost", State: LinkBroken},
	}); err != nil {
		t.Fatalf("AddLinks(p3) failed: %v", err)
	}

	broken, err := store.GetBrokenLinks()
	if err != nil {
		t.Fatalf("GetBrokenLinks failed: %v", err)
	}

	want := []BrokenLink{
		{FromPageID: "p2", FromTitle: "Beta", ToPath: "missing/alpha"},
		{FromPageID: "p1", FromTitle: "Alpha", ToPath: "missing/zeta"},
		{FromPageID: "p3", FromTitle: "Gamma", ToPath: "wikilink:Ghost"},
	}
	if len(broken) != len(want) {
		t.Fatalf("expected %d broken links, got %d: %+v", len(want), len(broken), broken)
	}
	for i, w := range want {
		if broken[i] != w {
			t.Errorf("broken[%d] = %+v, want %+v", i, broken[i], w)
		}
	}
}

func TestLinksStore_GetBrokenLinks_EmptyWhenNoneBroken(t *testing.T) {
	testutil.ResetMySQL(t)
	tmp := t.TempDir()
	store, err := NewLinksStore(tmp)
	if err != nil {
		t.Fatalf("NewLinksStore err: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)

	if err := store.AddLinks("p1", "Alpha", []TargetLink{
		{TargetPagePath: "exists/ok", State: LinkResolved},
	}); err != nil {
		t.Fatalf("AddLinks(p1) failed: %v", err)
	}

	broken, err := store.GetBrokenLinks()
	if err != nil {
		t.Fatalf("GetBrokenLinks failed: %v", err)
	}
	if len(broken) != 0 {
		t.Fatalf("expected no broken links, got %+v", broken)
	}
}

// A links.db from before the 3-state column (boolean `broken`) is dropped and
// recreated: the index is rebuilt from the markdown files on every startup.

// links.db is a derived index (ADR-0001): a corrupt file is deleted with its
// sidecars and rebuilt empty, unlike real-state stores (ADR-0013).
