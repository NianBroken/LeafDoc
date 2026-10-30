// Package tree stores page bodies, metadata and hierarchy in MySQL.
package tree

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/ignore"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
)

const nodeNamespace = "page_nodes"
const errEntryRequired = "a page is required"
const errParentEntryRequired = "a parent is required"

type NodeStore struct {
	executor    database.Executor
	storageDir  string
	log         *slog.Logger
	ignoreCache *ignore.Cache
}

// nodeRecord stores one node without children or parent pointers, avoiding recursive duplication.
type nodeRecord struct {
	Node     PageNode `json:"node"`
	ParentID string   `json:"parentId"`
	Raw      string   `json:"raw"`
}

func NewNodeStore(storageDir string) *NodeStore {
	return &NodeStore{storageDir: storageDir, log: slog.Default().With("component", "NodeStore")}
}
func (s *NodeStore) SetIgnoreCache(cache *ignore.Cache) { s.ignoreCache = cache }
func formatMetadataTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
func (s *NodeStore) syncManagedFrontmatter(md *markdown.MarkdownFile, n *PageNode) {
	md.SetLeafDocFrontmatter(n.ID, n.Title)
	md.SetLeafDocMetadata(formatMetadataTime(n.Metadata.CreatedAt), formatMetadataTime(n.Metadata.UpdatedAt), n.Metadata.CreatorID, n.Metadata.LastAuthorID)
	md.SetLeafDocPinned(n.Pinned)
}
func (s *NodeStore) readRecord(id string) (*nodeRecord, error) {
	raw, err := s.records().Read(nodeNamespace, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPageNotFound
	}
	if err != nil {
		return nil, err
	}
	var record nodeRecord
	if err = json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return &record, nil
}
func nodePayload(n *PageNode, raw string) ([]byte, error) {
	copy := *n
	copy.Children = nil
	copy.Parent = nil
	parent := "root"
	if n.Parent != nil {
		parent = n.Parent.ID
	}
	return json.Marshal(nodeRecord{Node: copy, ParentID: parent, Raw: raw})
}
func (s *NodeStore) writeRecord(n *PageNode, raw string) error {
	payload, err := nodePayload(n, raw)
	if err != nil {
		return err
	}
	return s.records().Write(nodeNamespace, n.ID, payload)
}
func (s *NodeStore) readMarkdown(n *PageNode) (*markdown.MarkdownFile, error) {
	record, err := s.readRecord(n.ID)
	if err != nil {
		return nil, err
	}
	return markdown.NewMarkdownFileFromRaw(n.Slug+".md", record.Raw)
}
func (s *NodeStore) writeMarkdown(n *PageNode, md *markdown.MarkdownFile) error {
	s.syncManagedFrontmatter(md, n)
	raw, err := markdown.BuildMarkdownWithFrontmatter(md.GetFrontmatter(), md.GetContent())
	if err != nil {
		return err
	}
	return s.writeRecord(n, raw)
}

// ReadPageRaw reads the authoritative document, including importable frontmatter.
func (s *NodeStore) ReadPageRaw(n *PageNode) (string, error) {
	if n == nil {
		return "", ErrPageNotFound
	}
	r, e := s.readRecord(n.ID)
	if e != nil {
		return "", e
	}
	return r.Raw, nil
}
func (s *NodeStore) ReadPageAndRaw(n *PageNode) (string, string, error) {
	raw, err := s.ReadPageRaw(n)
	if err != nil {
		return "", "", err
	}
	md, err := markdown.NewMarkdownFileFromRaw(n.Slug+".md", raw)
	if err != nil {
		return "", raw, err
	}
	return md.GetContent(), raw, nil
}
func (s *NodeStore) ReadPageContent(n *PageNode) (string, error) {
	body, _, err := s.ReadPageAndRaw(n)
	return body, err
}

// ReloadFromDatabase retains the public resync entry point while reading MySQL exclusively.
func (s *NodeStore) ReloadFromDatabase() (*PageNode, error) {
	return s.ReloadFromDatabaseContext(context.Background())
}
func (s *NodeStore) ReloadFromDatabaseContext(ctx context.Context) (*PageNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	records, err := s.records().List(nodeNamespace)
	if err != nil {
		return nil, err
	}
	root := &PageNode{ID: "root", Title: "root", Slug: "root", Kind: NodeKindSection, Children: []*PageNode{}}
	nodes := map[string]*PageNode{"root": root}
	parents := map[string]string{}
	for id, raw := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var r nodeRecord
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("decode page %s: %w", id, err)
		}
		if id == "root" {
			continue
		}
		n := r.Node
		n.Children = []*PageNode{}
		nodes[id] = &n
		parents[id] = r.ParentID
	}
	for id, parentID := range parents {
		n := nodes[id]
		p := nodes[parentID]
		if p == nil {
			return nil, fmt.Errorf("page %s has missing parent %s", id, parentID)
		}
		if p.Kind != NodeKindSection {
			return nil, fmt.Errorf("page %s has non-section parent", id)
		}
		n.Parent = p
		p.Children = append(p.Children, n)
	}
	visited := map[string]bool{}
	var sortTree func(*PageNode) error
	sortTree = func(n *PageNode) error {
		if visited[n.ID] {
			return fmt.Errorf("cyclic page hierarchy at %s", n.ID)
		}
		visited[n.ID] = true
		sort.SliceStable(n.Children, func(i, j int) bool {
			if n.Children[i].Position == n.Children[j].Position {
				return n.Children[i].ID < n.Children[j].ID
			}
			return n.Children[i].Position < n.Children[j].Position
		})
		slugs := map[string]bool{}
		for _, c := range n.Children {
			if slugs[c.Slug] {
				return ErrPageAlreadyExists
			}
			slugs[c.Slug] = true
			if err := sortTree(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := sortTree(root); err != nil {
		return nil, err
	}
	if len(visited) != len(nodes) {
		return nil, fmt.Errorf("page hierarchy contains unreachable nodes")
	}
	return root, nil
}

func (s *NodeStore) records() database.Records {
	return database.Records{Dir: s.storageDir, Executor: s.executor}
}
