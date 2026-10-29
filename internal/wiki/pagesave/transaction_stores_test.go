package pagesave

import (
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	"github.com/NianBroken/LeafDoc/internal/core/links"
	"github.com/NianBroken/LeafDoc/internal/core/properties"
	"github.com/NianBroken/LeafDoc/internal/core/search"
	"github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"log/slog"
	"testing"
)

// TestDerivedStoresShareCallerTransaction verifies internal batching never commits
// the outer page transaction and every derived record rolls back together.
func TestDerivedStoresShareCallerTransaction(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	db, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tagStore, err := tags.NewTagsStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer tagStore.Close()
	propertyStore, err := properties.NewPropertiesStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer propertyStore.Close()
	linkStore, err := links.NewLinksStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer linkStore.Close()
	index, err := search.NewMySQLIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	favoriteStore, err := favorites.NewFavoritesStore(dir, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	defer favoriteStore.Close()
	tables := []string{"page_tags", "page_meta", "page_properties", "links", "search_documents", "favorites"}
	for _, commit := range []bool{false, true} {
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		raw := "---\ntags: [transaction]\npriority: high\n---\n# Transaction\nshared record"
		operations := []func() error{
			func() error { return tags.NewTagsService(tagStore).WithExecutor(tx).IndexPageContent("page", raw) },
			func() error {
				return properties.NewPropertiesService(propertyStore).WithExecutor(tx).IndexPageContent("page", raw)
			},
			func() error {
				return linkStore.WithExecutor(tx).ReplaceLinksAndHeal([]links.PageLinkUpdate{{FromPageID: "page", FromTitle: "Transaction", ToPath: "page", Targets: []links.TargetLink{{TargetPageID: "target", TargetPagePath: "target"}}}})
			},
			func() error {
				return index.WithExecutor(tx).IndexPage("page", "page.md", "page", "Transaction", tree.NodeKindPage, raw)
			},
			func() error { return favoriteStore.WithExecutor(tx).Add("user", "page") },
		}
		for _, operation := range operations {
			if err := operation(); err != nil {
				t.Fatal(err)
			}
		}
		for _, table := range tables {
			var inside, outside int
			if err := tx.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&inside); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&outside); err != nil {
				t.Fatal(err)
			}
			if inside != 1 || outside != 0 {
				t.Fatalf("%s: transaction=%d pool=%d", table, inside, outside)
			}
		}
		if commit {
			err = tx.Commit()
		} else {
			err = tx.Rollback()
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, table := range tables {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
				t.Fatal(err)
			}
			expected := 0
			if commit {
				expected = 1
			}
			if count != expected {
				t.Fatalf("%s count=%d want=%d", table, count, expected)
			}
		}
	}
}
