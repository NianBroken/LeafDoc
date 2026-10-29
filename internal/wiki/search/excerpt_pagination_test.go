package search

import (
	"fmt"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"strings"
	"testing"

	coretags "github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

func setupExcerptPages(tb testing.TB, count int) (*SearchUseCase, []string) {
	tb.Helper()
	dir := tb.TempDir()
	store, err := coretags.NewTagsStore(dir)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := store.Close(); err != nil {
			tb.Error(err)
		}
	})
	tags := coretags.NewTagsService(store)
	svc := tree.NewTreeService(dir)
	if err := svc.LoadTree(); err != nil {
		tb.Fatal(err)
	}
	ids := make([]string, count)
	kind := tree.NodeKindPage
	for i := range ids {
		title := fmt.Sprintf("Page %04d", i)
		slug := fmt.Sprintf("page-%04d", i)
		id, err := svc.CreateNode("system", nil, title, slug, &kind)
		if err != nil {
			tb.Fatal(err)
		}
		ids[i] = *id
		raw := fmt.Sprintf("---\ntags: [docs]\n---\n\nUnique excerpt %04d", i)
		if err := svc.UpdateNode(tree.UpdateNodeInput{UserID: "system", ID: *id, Title: title, Slug: slug, Content: &raw, ExpectedVersion: tree.VersionUnchecked, PreserveFrontmatter: true}); err != nil {
			tb.Fatal(err)
		}
		page, err := svc.GetPage(*id)
		if err != nil {
			tb.Fatal(err)
		}
		if err := tags.IndexPageContent(*id, page.RawContent); err != nil {
			tb.Fatal(err)
		}
	}
	return NewSearchUseCase(nil, tags, svc), ids
}
func TestSearchUseCase_SearchByTags_PaginationPreservesExcerptsAndFacets(t *testing.T) {
	testutil.ResetMySQL(t)
	uc, ids := setupExcerptPages(t, 5)
	for _, tc := range []struct{ offset, limit, want int }{{1, 2, 2}, {10, 2, 0}, {-1, 0, 5}} {
		out, err := uc.searchByTags(ids, tc.offset, tc.limit)
		if err != nil {
			t.Fatal(err)
		}
		if out.Result.Count != 5 || len(out.Result.Items) != tc.want {
			t.Fatalf("got %#v", out.Result)
		}
		if len(out.Result.TagFacets) != 1 || out.Result.TagFacets[0].Count != 5 {
			t.Fatalf("facets %#v", out.Result.TagFacets)
		}
		for _, item := range out.Result.Items {
			suffix := strings.TrimPrefix(item.Title, "Page ")
			if !strings.Contains(item.Excerpt, "Unique excerpt "+suffix) {
				t.Fatalf("wrong excerpt %+v", item)
			}
		}
	}
}
func BenchmarkSearchByTagsPagedExcerpts(b *testing.B) {
	for _, count := range []int{100, 1000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			uc, ids := setupExcerptPages(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := uc.searchByTags(ids, 0, 20); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
