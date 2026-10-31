package tree

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
)

// SaveChildOrder 按父节点当前 Children 顺序保存各子节点现有字段，正文沿用数据库记录。
// 此方法不重新分配 Position；实际事务边界由 BeginExecutor 对连接类型的处理确定。
// 写入失败通过延迟回滚处理，成功后提交，父节点本身不写入。
func (s *NodeStore) SaveChildOrder(parent *PageNode) (resultErr error) {
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
	defer database.Rollback(tx, &resultErr)
	// 读取沿用当前存储执行器，写入使用取得的事务执行器，二者不在此重新绑定。
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

// create 验证父节点存在且为分区，再构造节点初始正文并保存。
// 普通页面以节点标题生成一级标题，分区正文为空；父子关系使用 n 当前的 Parent。
// 传入 parent 只参与入口验证，不在此改写节点的父指针或父节点 Children。
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

// CreatePage 委托统一建立流程，实际保存的类型由节点 Kind 字段确定。
func (s *NodeStore) CreatePage(parent, n *PageNode) error { return s.create(parent, n) }

// CreateSection 与页面建立共用存储流程，不在此强制改写节点类型。
func (s *NodeStore) CreateSection(parent, n *PageNode) error { return s.create(parent, n) }

// UpsertContent 要求页面记录已存在，读取原元信息后替换正文并同步节点管理字段。
func (s *NodeStore) UpsertContent(n *PageNode, body string) error {
	md, err := s.readMarkdown(n)
	if err != nil {
		return err
	}
	md.SetContent(body)
	return s.writeMarkdown(n, md)
}

// UpsertContentPreservingFrontmatter 将完整输入交给 Markdown 的管理字段保留规则处理。
// 解析成功后仍以节点当前状态同步系统字段，再保存完整页面记录。
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

// UpsertContentAndMetadata 替换正文，并按传入属性和标签重建自定义元信息。
// 原字符串与字符串键映射字段不自动保留，其余非 tags 字段保留后再写入 properties。
// tags 为 nil 时保留原标签，非 nil 时使用传入列表，空列表也会明确替换原标签。
func (s *NodeStore) UpsertContentAndMetadata(n *PageNode, body string, tags []string, properties map[string]string) error {
	md, err := s.readMarkdown(n)
	if err != nil {
		return err
	}
	md.SetContent(body)
	existing := md.GetFrontmatter().ExtraFields
	extra := map[string]interface{}{}
	// 保留规则按实际 Go 值类型判断，不递归检查映射内部的数据。
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

// MoveNode 保留数据库正文，用节点副本和指定父指针重写父标识。
// 不修改内存树关系，不校验目标分区、循环或同级路径段冲突，调用方负责这些条件。
func (s *NodeStore) MoveNode(n, parent *PageNode) error {
	r, err := s.readRecord(n.ID)
	if err != nil {
		return err
	}
	copy := *n
	copy.Parent = parent
	return s.writeRecord(&copy, r.Raw)
}

// DeletePage 共用子树删除流程，删除范围由传入节点的 Children 决定。
func (s *NodeStore) DeletePage(n *PageNode) error { return s.DeleteSection(n) }

// DeleteSection 按内存中的子树递归删除数据库记录，先删子节点再删当前节点。
// 执行器通过 BeginExecutor 获取，事务能力沿用其连接类型规则。
// 此处不删除附件或修改内存树，也不重新查询数据库中未出现在 Children 的后代。
func (s *NodeStore) DeleteSection(n *PageNode) (resultErr error) {
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
	defer database.Rollback(tx, &resultErr)
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

// RenameNode 用节点副本保存指定路径段，原全文和原内存节点保持不变。
// 名称有效性与冲突由调用方检查，存储入口只读原记录再写回。
func (s *NodeStore) RenameNode(n *PageNode, slug string) error {
	r, err := s.readRecord(n.ID)
	if err != nil {
		return err
	}
	copy := *n
	copy.Slug = slug
	return s.writeRecord(&copy, r.Raw)
}

// SyncFrontmatterIfExists 读取现有正文并重新同步节点系统字段。
// 记录不存在时返回读取错误，不因方法名中的 IfExists 而静默跳过。
func (s *NodeStore) SyncFrontmatterIfExists(n *PageNode) error {
	md, err := s.readMarkdown(n)
	if err != nil {
		return err
	}
	return s.writeMarkdown(n, md)
}

// ConvertNode 只接受页面或分区类型，有子节点时禁止转为页面。
// 使用节点副本保存类型并保留正文，不在此改变内存节点或进行标题重建。
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

// SetPinnedFrontmatter 用节点副本保存置顶字段并返回正文，不直接改变原节点。
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

// UpdateRecord 通过 FOR UPDATE 读取持久化版本，检查预期版本后保存节点和正文。
// 返回节点以传入 n 为基础复制，标题、路径段、更新时间和最后作者由输入及当前时间设定。
// 提交成功才返回副本，内存树和索引由调用方发布；事务隔离沿用 BeginExecutor 规则。
func (s *NodeStore) UpdateRecord(n *PageNode, in UpdateNodeInput) (result *PageNode, resultErr error) {
	db, err := s.records().Connection()
	if err != nil {
		return nil, err
	}
	tx, err := database.BeginExecutor(db)
	if err != nil {
		return nil, err
	}
	defer database.Rollback(tx, &resultErr)
	var payload []byte
	if err = tx.QueryRow("SELECT payload FROM content_records WHERE namespace=? AND record_key=? FOR UPDATE", nodeNamespace, n.ID).Scan(&payload); err != nil {
		return nil, err
	}
	var record nodeRecord
	if err = json.Unmarshal(payload, &record); err != nil {
		return nil, err
	}
	// 并发版本检查使用刚读取的数据库节点，而不是调用方传入对象中的时间字段。
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
	// 未提供正文时保留数据库全文，仅同步节点管理字段。
	// 完整元信息保留模式与独立标签属性模式是两个互斥分支。
	if in.Content != nil {
		if in.PreserveFrontmatter {
			if err = md.SetRawContentPreservingManagedFrontmatter(*in.Content); err != nil {
				return nil, err
			}
		} else {
			md.SetContent(*in.Content)
			// 至少提供一个元信息集合才重建附加字段；nil 与空集合具有不同含义。
			if in.Tags != nil || in.Properties != nil {
				existing := md.GetFrontmatter().ExtraFields
				extra := map[string]interface{}{}
				// 原字符串和映射属性不自动保留，其余类型字段先复制，再写传入属性。
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
	// 系统字段统一从待提交节点取得，避免外部正文携带的管理值覆盖节点身份。
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
