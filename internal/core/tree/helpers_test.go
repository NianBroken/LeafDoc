package tree

import (
	"errors"
	"testing"
	"time"
)

func TestNodeVersionRules(t *testing.T) {
	n := &PageNode{Metadata: PageMetadata{UpdatedAt: time.Now().UTC()}}
	if err := checkNodeVersion(n, ""); !errors.Is(err, ErrVersionRequired) {
		t.Fatal(err)
	}
	if err := checkNodeVersion(n, "old"); !errors.Is(err, ErrVersionConflict) {
		t.Fatal(err)
	}
	if err := checkNodeVersion(n, n.Version()); err != nil {
		t.Fatal(err)
	}
	if err := checkNodeVersion(n, VersionUnchecked); err != nil {
		t.Fatal(err)
	}
}
func TestLogicalPaths(t *testing.T) {
	root := &PageNode{ID: "root", Slug: "root"}
	a := &PageNode{Slug: "docs", Parent: root}
	b := &PageNode{Slug: "guide", Parent: a}
	if got := GeneratePathFromPageNode(b); got != "root/docs/guide" {
		t.Fatal(got)
	}
}
