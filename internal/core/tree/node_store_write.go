package tree

import (
	"encoding/json"
	"fmt"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
	"time"
)

// SaveChildOrder persists all sibling positions in a single transaction.
func (s *NodeStore) SaveChildOrder(parent *PageNode) error {
	if parent == nil {
		return &InvalidOpError{Op: "SaveChildOrder", Reason: errParentEntryRequired}
	}
	db, err := s.records().Connection()
	if err != nil {
		return err
	}
	tx, err := database.BeginExecutor(db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, child := range parent.Children {
		r, err := s.readRecord(child.ID)
		if err != nil {
			return err
		}
		raw, err := nodePayload(child, r.Raw)
		if err != nil {
			return err
		}
		if _, err = tx.Exec("UPDATE content_records SET payload=? WHERE namespace=? AND record_key=?", raw, nodeNamespace, child.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *NodeStore) create(parent, n *PageNode) error {
	if parent == nil || n == nil {
		return &InvalidOpError{Op: "Create", Reason: errEntryRequired}
	}
	if parent.Kind != NodeKindSection {
		return &InvalidOpError{Op: "Create", Reason: "parent must be a section"}
	}
	body := "# " + n.Title + "\n"
	if n.Kind == NodeKindSection {
		body = ""
	}
	md := markdown.NewMarkdownFile(n.Slug+".md", body, markdown.Frontmatter{})
	return s.writeMarkdown(n, md)
}
func (s *NodeStore) CreatePage(parent, n *PageNode) error    { return s.create(parent, n) }
func (s *NodeStore) CreateSection(parent, n *PageNode) error { return s.create(parent, n) }
func (s *NodeStore) UpsertContent(n *PageNode, body string) error {
	md, err := s.readMarkdown(n)
	if err != nil {
		return err
	}
	md.SetContent(body)
	return s.writeMarkdown(n, md)
}
func (s *NodeStore) UpsertContentPreservingFrontmatter(n *PageNode, raw string) error {
	md, err := s.readMarkdown(n)
	if err != nil {
		return err
	}
	if err = md.SetRawContentPreservingManagedFrontmatter(raw); err != nil {
		return err
	}
	return s.writeMarkdown(n, md)
}
func (s *NodeStore) UpsertContentAndMetadata(n *PageNode, body string, tags []string, properties map[string]string) error {
	md, err := s.readMarkdown(n)
	if err != nil {
		return err
	}
	md.SetContent(body)
	existing := md.GetFrontmatter().ExtraFields
	extra := map[string]interface{}{}
	for k, v := range existing {
		if k == "tags" {
			continue
		}
		switch v.(type) {
		case string, map[string]interface{}:
		default:
			extra[k] = v
		}
	}
	for k, v := range properties {
		extra[k] = v
	}
	if tags != nil {
		values := make([]interface{}, len(tags))
		for i, t := range tags {
			values[i] = t
		}
		extra["tags"] = values
	} else if v, ok := existing["tags"]; ok {
		extra["tags"] = v
	}
	md.SetExtraFields(extra)
	return s.writeMarkdown(n, md)
}
func (s *NodeStore) MoveNode(n, parent *PageNode) error {
	r, err := s.readRecord(n.ID)
	if err != nil {
		return err
	}
	copy := *n
	copy.Parent = parent
	return s.writeRecord(&copy, r.Raw)
}
func (s *NodeStore) DeletePage(n *PageNode) error { return s.DeleteSection(n) }

// DeleteSection removes the complete subtree in one transaction.
func (s *NodeStore) DeleteSection(n *PageNode) error {
	if n == nil {
		return ErrPageNotFound
	}
	db, err := s.records().Connection()
	if err != nil {
		return err
	}
	tx, err := database.BeginExecutor(db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var remove func(*PageNode) error
	remove = func(node *PageNode) error {
		for _, child := range node.Children {
			if err := remove(child); err != nil {
				return err
			}
		}
		_, err := tx.Exec("DELETE FROM content_records WHERE namespace=? AND record_key=?", nodeNamespace, node.ID)
		return err
	}
	if err = remove(n); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *NodeStore) RenameNode(n *PageNode, slug string) error {
	r, err := s.readRecord(n.ID)
	if err != nil {
		return err
	}
	copy := *n
	copy.Slug = slug
	return s.writeRecord(&copy, r.Raw)
}
func (s *NodeStore) SyncFrontmatterIfExists(n *PageNode) error {
	md, err := s.readMarkdown(n)
	if err != nil {
		return err
	}
	return s.writeMarkdown(n, md)
}
func (s *NodeStore) ConvertNode(n *PageNode, target NodeKind) error {
	if target != NodeKindPage && target != NodeKindSection {
		return fmt.Errorf("invalid node kind %s", target)
	}
	if target == NodeKindPage && len(n.Children) > 0 {
		return ErrPageHasChildren
	}
	md, err := s.readMarkdown(n)
	if err != nil {
		return err
	}
	copy := *n
	copy.Kind = target
	return s.writeMarkdown(&copy, md)
}
func (s *NodeStore) SetPinnedFrontmatter(n *PageNode, pinned bool) (string, error) {
	md, err := s.readMarkdown(n)
	if err != nil {
		return "", err
	}
	copy := *n
	copy.Pinned = pinned
	if err = s.writeMarkdown(&copy, md); err != nil {
		return "", err
	}
	return md.GetContent(), nil
}

// UpdateRecord locks the persisted version and commits content plus metadata atomically.
// The caller publishes the returned node to its in-memory indexes after commit.
func (s *NodeStore) UpdateRecord(n *PageNode, in UpdateNodeInput) (*PageNode, error) {
	db, err := s.records().Connection()
	if err != nil {
		return nil, err
	}
	tx, err := database.BeginExecutor(db)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var payload []byte
	if err = tx.QueryRow("SELECT payload FROM content_records WHERE namespace=? AND record_key=? FOR UPDATE", nodeNamespace, n.ID).Scan(&payload); err != nil {
		return nil, err
	}
	var record nodeRecord
	if err = json.Unmarshal(payload, &record); err != nil {
		return nil, err
	}
	if err = checkNodeVersion(&record.Node, in.ExpectedVersion); err != nil {
		return nil, err
	}
	updated := *n
	updated.Title = in.Title
	updated.Slug = in.Slug
	updated.Metadata.UpdatedAt = time.Now().UTC()
	updated.Metadata.LastAuthorID = in.UserID
	md, err := markdown.NewMarkdownFileFromRaw(updated.Slug+".md", record.Raw)
	if err != nil {
		return nil, err
	}
	if in.Content != nil {
		if in.PreserveFrontmatter {
			if err = md.SetRawContentPreservingManagedFrontmatter(*in.Content); err != nil {
				return nil, err
			}
		} else {
			md.SetContent(*in.Content)
			if in.Tags != nil || in.Properties != nil {
				existing := md.GetFrontmatter().ExtraFields
				extra := map[string]interface{}{}
				for key, value := range existing {
					if key == "tags" {
						continue
					}
					switch value.(type) {
					case string, map[string]interface{}:
					default:
						extra[key] = value
					}
				}
				for key, value := range in.Properties {
					extra[key] = value
				}
				if in.Tags != nil {
					tags := make([]interface{}, len(in.Tags))
					for i, v := range in.Tags {
						tags[i] = v
					}
					extra["tags"] = tags
				} else if tags, ok := existing["tags"]; ok {
					extra["tags"] = tags
				}
				md.SetExtraFields(extra)
			}
		}
	}
	s.syncManagedFrontmatter(md, &updated)
	raw, err := markdown.BuildMarkdownWithFrontmatter(md.GetFrontmatter(), md.GetContent())
	if err != nil {
		return nil, err
	}
	payload, err = nodePayload(&updated, raw)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec("UPDATE content_records SET payload=? WHERE namespace=? AND record_key=?", payload, nodeNamespace, n.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &updated, nil
}
