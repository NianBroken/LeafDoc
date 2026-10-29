package tree

import (
	"strings"
	"testing"
)

// Imported custom frontmatter and literal editor body text retain distinct semantics.
func TestMySQLNodeFrontmatterAndPinning(t *testing.T) {
	s, _ := newLoadedService(t)
	id := createTestNode(t, s, nil, "文档", "guide", NodeKindPage)
	n := s.getNodeByIDLocked(id)
	if err := s.store.UpsertContentPreservingFrontmatter(n, "---\ntags: [教程]\nstatus: published\n---\n# 内容\n"); err != nil {
		t.Fatal(err)
	}
	raw, err := s.store.ReadPageRaw(n)
	if err != nil || !strings.Contains(raw, "published") {
		t.Fatalf("frontmatter %s %v", raw, err)
	}
	literal := "---\nnot: metadata\n---\n正文"
	if err = s.store.UpsertContent(n, literal); err != nil {
		t.Fatal(err)
	}
	body, err := s.store.ReadPageContent(n)
	if err != nil || body != literal {
		t.Fatalf("literal body %q %v", body, err)
	}
	if _, err = s.SetPinned(id, n.Version(), true); err != nil {
		t.Fatal(err)
	}
	if err = s.LoadTree(); err != nil {
		t.Fatal(err)
	}
	p, err := s.GetPage(id)
	if err != nil || !p.Pinned {
		t.Fatal("pinned state missing", err)
	}
}
