package tree

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// createNodeResult 保存建立节点的结果及父节点类型转换状态，供失败清理使用。
type createNodeResult struct {
	id                 string
	entry              *PageNode
	parent             *PageNode
	parentWasConverted bool
}

// CreateNode 通过统一写锁入口建立节点，成功返回页面标识。
// 路径段由调用方传入，此入口检查同级冲突，不调用名称生成器重新规范处理。
func (t *TreeService) CreateNode(userID string, parentID *string, title string, slug string, nodeKind *NodeKind) (*string, error) {
	var result *string
	err := t.withLockedTree(func() error {
		created, err := t.createNodeLocked(userID, parentID, title, slug, nodeKind)
		if err != nil {
			return err
		}
		result = &created.id

		return nil
	})

	return result, err
}

// createNodeLocked 在已持有树写锁时建立节点，先保存记录，再发布到父节点与索引。
// 保存子顺序失败时尝试删除已建立的节点，并按条件恢复父节点类型。
// 父节点转换、标识生成和节点建立不在此组成单一自动回滚事务。
func (t *TreeService) createNodeLocked(userID string, parentID *string, title string, slug string, kind *NodeKind) (*createNodeResult, error) {
	if t.tree == nil {
		return nil, ErrTreeNotLoaded
	}

	// 未指定类型时使用普通页面，指定值在后面的存储分支中选择写入入口。
	k := NodeKindPage
	if kind != nil {
		k = *kind
	}

	// 空父标识和 root 使用虚拟根，其余标识必须能在当前树中找到。
	parent := t.tree
	if parentID != nil && *parentID != "" && *parentID != "root" {
		parent = t.getNodeByIDLocked(*parentID)
		if parent == nil {
			return nil, ErrParentNotFound
		}
	}

	// 同级路径段已经存在时拒绝建立，不自动添加数字后缀。
	if t.findChildBySlugInParentLocked(parent, slug) != nil {
		return nil, ErrPageAlreadyExists
	}

	parentWasConverted := false

	// 非根父节点若不是分区，先在存储中转为分区，再改变内存类型。
	// 后续步骤失败时，只有进入对应清理路径才会尝试恢复这个类型。
	if parent.Kind != NodeKindSection && parent.ID != "root" {
		t.log.Info("converting parent to section", "parentID", parent.ID, "oldKind", parent.Kind, "newKind", NodeKindSection)
		if err := t.store.ConvertNode(parent, NodeKindSection); err != nil {
			return nil, fmt.Errorf("could not convert parent node: %w", err)
		}
		parent.Kind = NodeKindSection
		parentWasConverted = true
	}

	if parent.Kind != NodeKindSection {
		return nil, fmt.Errorf("cannot add child to non-section parent, got %q", parent.Kind)
	}

	id, err := shared.GenerateUniqueID()
	if err != nil {
		return nil, fmt.Errorf("could not generate unique ID: %w", err)
	}

	now := time.Now().UTC()

	entry := &PageNode{
		ID:       id,
		Title:    title,
		Parent:   parent,
		Slug:     slug,
		Kind:     k,
		Position: len(parent.Children), // 初始位置取已有子节点数，放在列表尾部。
		Children: []*PageNode{},
		Metadata: PageMetadata{
			CreatedAt:    now,
			UpdatedAt:    now,
			CreatorID:    userID,
			LastAuthorID: userID,
		},
	}

	// 支持的页面和分区类型分别调用存储入口，错误返回时尚未发布子节点关系。
	switch k {
	case NodeKindPage:
		if err := t.store.CreatePage(parent, entry); err != nil {
			return nil, fmt.Errorf("could not create page entry: %w", err)
		}
	case NodeKindSection:
		if err := t.store.CreateSection(parent, entry); err != nil {
			return nil, fmt.Errorf("could not create section entry: %w", err)
		}
	}

	// 记录保存成功后发布父子关系和索引，再保存父节点的子顺序。
	parent.Children = append(parent.Children, entry)
	t.indexNodeLocked(entry)
	if err := t.store.SaveChildOrder(parent); err != nil {
		rollbackErr := t.rollbackCreatedNodeLocked(parent, entry, parentWasConverted)
		if rollbackErr != nil {
			return nil, errors.Join(fmt.Errorf(errPersistChildOrderFailed, err), fmt.Errorf("rollback created node: %w", rollbackErr))
		}
		return nil, fmt.Errorf(errPersistChildOrderFailed, err)
	}
	return &createNodeResult{
		id:                 entry.ID,
		entry:              entry,
		parent:             parent,
		parentWasConverted: parentWasConverted,
	}, nil
}

// rollbackCreatedNodeLocked 在树锁内撤销已发布节点，先移除内存关系和索引再删除记录。
// 只有父节点因该建立过程转为分区且已经没有子节点，才尝试转回普通页面。
// 删除或转换失败直接返回，调用方将该错误与原失败组合。
func (t *TreeService) rollbackCreatedNodeLocked(parent *PageNode, entry *PageNode, parentWasConverted bool) error {
	if parent == nil || entry == nil {
		return nil
	}

	for i, child := range parent.Children {
		if child == entry {
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
			break
		}
	}
	t.removeNodeIndexLocked(entry)

	switch entry.Kind {
	case NodeKindSection:
		if err := t.store.DeleteSection(entry); err != nil {
			return err
		}
	case NodeKindPage:
		if err := t.store.DeletePage(entry); err != nil {
			return err
		}
	}

	if parentWasConverted && len(parent.Children) == 0 {
		if err := t.store.ConvertNode(parent, NodeKindPage); err != nil {
			return err
		}
		parent.Kind = NodeKindPage
	}

	return nil
}

// DeleteNode 检查页面版本与递归许可，删除存储记录后移除内存子树并保存剩余顺序。
// 删除后的顺序保存失败不会在此重建已删除的子树，错误交给调用方处理。
func (t *TreeService) DeleteNode(userID string, id string, recursive bool, expectedVersion string) error {
	err := t.withLockedTree(func() error {
		if t.tree == nil {
			return ErrTreeNotLoaded
		}

		// 先查找节点并校验版本，根节点缺少父指针时不能继续删除。
		node := t.getNodeByIDLocked(id)
		if node == nil {
			return ErrPageNotFound
		}

		if err := checkNodeVersion(node, expectedVersion); err != nil {
			return err
		}

		// 非递归删除禁止处理有子节点的目标，无论目标声明为何种节点类型。
		if node.HasChildren() && !recursive {
			return ErrPageHasChildren
		}

		// 保留父节点供记录删除成功后断开关系，当前尚未改变内存树。
		parent := node.Parent
		if parent == nil {
			return ErrParentNotFound
		}

		switch node.Kind {
		case NodeKindSection:
			if err := t.store.DeleteSection(node); err != nil {
				return fmt.Errorf("could not delete section entry: %w", err)
			}
		case NodeKindPage:
			if node.HasChildren() {
				// 递归许可下，带子节点的普通页面先转换成分区，再按子树删除。
				t.log.Info("converting page to section for recursive delete", "pageID", node.ID)
				if err := t.store.ConvertNode(node, NodeKindSection); err != nil {
					return fmt.Errorf("could not convert page to section: %w", err)
				}
				node.Kind = NodeKindSection
				if err := t.store.DeleteSection(node); err != nil {
					return fmt.Errorf("could not delete section entry: %w", err)
				}
			} else {
				if err := t.store.DeletePage(node); err != nil {
					return fmt.Errorf("could not delete page entry: %w", err)
				}
			}
		default:
			return fmt.Errorf("unknown node kind: %v", node.Kind)
		}

		// 存储删除完成后，从父列表和索引中移除节点及其后代。
		for i, e := range parent.Children {
			if e.ID == id {
				parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
				break
			}
		}
		t.removeNodeIndexLocked(node)

		t.reindexPositions(parent)
		if err := t.store.SaveChildOrder(parent); err != nil {
			return fmt.Errorf(errPersistChildOrderFailed, err)
		}
		return nil
	})
	return err
}

// UpdateNodeInput 提供节点标题、路径段、预期版本和可选正文。
// Content 为空指针时保留正文，Tags 与 Properties 也不进入处理分支。
// PreserveFrontmatter 为 true 时正文按完整 Markdown 处理，独立标签和属性不使用。
// 其余模式将 Content 作为纯正文，并在至少一个元信息集合非 nil 时重建附加字段。
type UpdateNodeInput struct {
	UserID              string
	ID                  string
	Title               string
	Slug                string
	Content             *string
	ExpectedVersion     string
	Tags                []string
	Properties          map[string]string
	PreserveFrontmatter bool
}

// UpdateNode 在树锁内检查当前版本和同级名称冲突，再由存储层校验持久化版本并提交。
// 提交成功后覆盖原节点字段，重建标题索引和父节点路径段索引，保持节点指针身份。
func (t *TreeService) UpdateNode(in UpdateNodeInput) error {
	return t.withLockedTree(func() error {
		if t.tree == nil {
			return ErrTreeNotLoaded
		}
		node := t.getNodeByIDLocked(in.ID)
		if node == nil {
			return ErrPageNotFound
		}
		if err := checkNodeVersion(node, in.ExpectedVersion); err != nil {
			return err
		}
		if in.Slug != node.Slug && node.Parent != nil {
			existing := t.findChildBySlugInParentLocked(node.Parent, in.Slug)
			if existing != nil && existing.ID != node.ID {
				return ErrPageAlreadyExists
			}
		}
		updated, err := t.store.UpdateRecord(node, in)
		if err != nil {
			return err
		}
		t.removeTitleIndexForNodeLocked(node)
		*node = *updated
		t.addTitleIndexForNodeLocked(node)
		if node.Parent != nil {
			t.rebuildChildSlugIndexForParentLocked(node.Parent)
		}
		return nil
	})
}

// ConvertNode 校验版本后改变节点类型，同类型请求不写入也不更新时间。
// 类型先由存储层保存，再更新内存时间和作者并同步全文元信息。
// 后一步同步失败直接返回错误，不在此撤销前一步类型保存。
func (t *TreeService) ConvertNode(userID string, id string, kind NodeKind, expectedVersion string) error {
	return t.withLockedTree(func() error {
		if t.tree == nil {
			return ErrTreeNotLoaded
		}

		// 仅操作已加载树中的节点，不根据给定标识自动创建页面。
		node := t.getNodeByIDLocked(id)
		if node == nil {
			return ErrPageNotFound
		}

		if err := checkNodeVersion(node, expectedVersion); err != nil {
			return err
		}

		if node.Kind == kind {
			// 目标类型已满足，版本检查仍已在前面执行。
			return nil
		}

		// 分区有子节点时不能转为普通页面，避免丢失层级容纳关系。
		if node.Kind == NodeKindSection && kind == NodeKindPage && node.HasChildren() {
			return ErrPageHasChildren
		}

		t.log.Info("changing node kind", "nodeID", node.ID, "oldKind", node.Kind, "newKind", kind)

		if err := t.store.ConvertNode(node, kind); err != nil {
			return fmt.Errorf("could not convert node: %w", err)
		}
		node.Kind = kind

		// 类型保存成功后在内存中记录修改时间和操作用户。
		node.Metadata.UpdatedAt = time.Now().UTC()
		node.Metadata.LastAuthorID = userID

		// 将内存中的类型与元信息再次同步到完整数据库页面记录。
		if err := t.store.SyncFrontmatterIfExists(node); err != nil {
			return fmt.Errorf("could not sync frontmatter: %w", err)
		}

		// 各项存储操作已完成，此处直接结束写锁流程。
		return nil
	})
}

// BulkContentUpdate 为批量写入提供页面标识与纯正文，不包含版本或独立元信息。
type BulkContentUpdate struct {
	ID      string
	Content string
}

// BulkUpdateContent 在单次树写锁内处理全部输入，返回与输入位置对应的错误列表。
// 页面正文、更新时间和最后作者参与写入，标题与路径段沿用节点当前值。
// 工作单元内串行写入，其余情况每项启动一个协程，等待全部结束后释放树锁。
func (t *TreeService) BulkUpdateContent(userID string, updates []BulkContentUpdate) []error {
	errs := make([]error, len(updates))
	if len(updates) == 0 {
		return errs
	}

	type task struct {
		index       int
		node        *PageNode
		content     string
		oldMetadata PageMetadata
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tree == nil {
		for i := range errs {
			errs[i] = ErrTreeNotLoaded
		}
		return errs
	}

	now := time.Now().UTC()
	tasks := make([]task, 0, len(updates))
	for i, u := range updates {
		node := t.getNodeByIDLocked(u.ID)
		if node == nil {
			errs[i] = ErrPageNotFound
			continue
		}
		oldMetadata := node.Metadata
		// 全批次使用同一时间，保存前先设置内存元信息，并记录失败时使用的原值。
		node.Metadata.UpdatedAt = now
		node.Metadata.LastAuthorID = userID
		tasks = append(tasks, task{index: i, node: node, content: u.Content, oldMetadata: oldMetadata})
	}

	if len(tasks) == 0 {
		return errs
	}

	// 同一工作单元共用执行器，逐项处理而不并发调用该执行器。
	if t.InUnitOfWork() {
		for _, tk := range tasks {
			errs[tk.index] = t.store.UpsertContent(tk.node, tk.content)
			if errs[tk.index] != nil {
				tk.node.Metadata = tk.oldMetadata
			}
		}
		return errs
	}

	// 非工作单元路径并发写各项，用独立互斥锁保护错误回写，树写锁继续保持。
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for _, tk := range tasks {
		go func(tk task) {
			defer wg.Done()
			if err := t.store.UpsertContent(tk.node, tk.content); err != nil {
				mu.Lock()
				errs[tk.index] = err
				mu.Unlock()
			}
		}(tk)
	}
	wg.Wait()

	// 所有写入结束后仅恢复失败项的内存元信息，不撤销其他项已成功的写入。
	for _, tk := range tasks {
		if errs[tk.index] != nil {
			tk.node.Metadata = tk.oldMetadata
		}
	}

	return errs
}

// EnsurePagePath 在树写锁内沿解析路径建立缺失节点，返回最终节点和已建立节点的摘要。
// 中间节点使用分区，末段未指定类型时也保持分区，不采用 createNodeLocked 的页面默认值。
// 已存在完整路径直接返回现有节点，不覆盖标题或类型；中途失败不撤销之前已建立的段。
func (t *TreeService) EnsurePagePath(userID string, p string, targetTitle string, kind *NodeKind) (*EnsurePathResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tree == nil {
		return nil, ErrTreeNotLoaded
	}

	created := []*PageNode{}

	lookup, err := t.lookupPagePathLocked(p)
	if err != nil {
		return nil, fmt.Errorf("could not lookup page path: %w", err)
	}

	// 路径全部存在时用查询末段标识读取现有节点，不执行正文或元信息写入。
	if lookup.Exists {
		last := lookup.Segments[len(lookup.Segments)-1]
		page := t.getNodeByIDLocked(*last.ID)
		if page == nil {
			return nil, fmt.Errorf("could not find existing page by ID: %w", ErrPageNotFound)
		}
		return &EnsurePathResult{Exists: true, Page: page}, nil
	}

	// 按路径顺序推进父标识，存在的段直接经过，仅为缺失段执行创建。
	var currentID *string // 空父标识由创建入口解释为虚拟根节点。
	for i, segment := range lookup.Segments {
		if segment.Exists {
			currentID = segment.ID
			continue
		}

		// 中间分区以路径段为标题，末段才使用调用方给定的目标标题。
		segTitle := segment.Slug
		if i == len(lookup.Segments)-1 {
			segTitle = targetTitle
		}

		// 所有段默认分区，仅末段能由显式 kind 覆盖。
		kindToUse := NodeKindSection
		if i == len(lookup.Segments)-1 && kind != nil {
			kindToUse = *kind
		}

		createdNode, err := t.createNodeLocked(userID, currentID, segTitle, segment.Slug, &kindToUse)
		if err != nil {
			return nil, fmt.Errorf("could not create segment %q: %w", segment.Slug, err)
		}
		currentID = &createdNode.id

		created = append(created, &PageNode{
			ID:    createdNode.id,
			Slug:  segment.Slug,
			Title: segTitle,
			Kind:  kindToUse,
		})
	}

	// 最终页面返回当前树中的真实对象，Created 列表只包含上面组装的有限字段。
	if currentID == nil {
		return nil, fmt.Errorf("could not ensure page path")
	}
	page := t.getNodeByIDLocked(*currentID)
	if page == nil {
		return nil, fmt.Errorf("could not find created page by ID: %w", ErrPageNotFound)
	}

	// 各段创建时已分别持久化，此处只组装结果，不另行整体保存。
	return &EnsurePathResult{
		Exists:  true,
		Page:    page,
		Created: created,
	}, nil
}

// MoveNode 使用负位置调用移动入口，目标父节点可由空串或 root 指定为根。
// 同父节点且位置为负会被移动入口拒绝，其他移动沿用其位置重排规则。
func (t *TreeService) MoveNode(userID string, id string, parentID string, expectedVersion string) error {
	return t.MoveNodeToPosition(userID, id, parentID, expectedVersion, -1)
}

// MoveNodeToPosition 检查版本、同级冲突与循环后保存父变更，并更新内存父子关系。
// position 用于选择插入位置，负数或越界先选择末尾；随后仍按 Position 稳定重排。
// 保存来源顺序、目标顺序或节点元信息失败时，调用清理入口尝试恢复先前状态。
func (t *TreeService) MoveNodeToPosition(userID string, id string, parentID string, expectedVersion string, position int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tree == nil {
		return ErrTreeNotLoaded
	}

	// 移动节点必须存在并具有有效父节点，根节点不能通过此路径移动。
	node := t.getNodeByIDLocked(id)
	if node == nil {
		return ErrPageNotFound
	}

	if err := checkNodeVersion(node, expectedVersion); err != nil {
		return err
	}

	oldParent := node.Parent
	if oldParent == nil {
		return fmt.Errorf("old parent not found: %w", ErrParentNotFound)
	}

	// 目标父标识为空或 root 时使用虚拟根，其余必须能按标识找到。
	newParent := t.tree
	if parentID != "" && parentID != "root" {
		newParent = t.getNodeByIDLocked(parentID)
		if newParent == nil {
			return fmt.Errorf("new parent not found: %w", ErrParentNotFound)
		}
	}

	// 同父且没有明确位置视为无效操作，其他情况仍须检查名称是否被其他节点占用。
	if newParent.ID == oldParent.ID && position < 0 {
		return &InvalidOpError{Op: "MoveNode", Reason: "page already belongs to the destination parent"}
	}
	if existing := t.findChildBySlugInParentLocked(newParent, node.Slug); existing != nil && existing.ID != node.ID {
		return fmt.Errorf("child with the same slug already exists: %w", ErrPageAlreadyExists)
	}

	// 节点不能作为自身的父节点。
	if node.ID == newParent.ID {
		return fmt.Errorf("page cannot be moved to itself: %w", ErrPageCannotBeMovedToItself)
	}

	// IsChildOf 在接收者的后代中查找目标标识，命中说明新父节点位于当前子树内。
	if node.IsChildOf(newParent.ID, true) {
		return fmt.Errorf("circular reference detected: %w", ErrMovePageCircularReference)
	}

	newParentWasConverted := false
	if newParent.ID != "root" && newParent.Kind == NodeKindPage {
		if err := t.store.ConvertNode(newParent, NodeKindSection); err != nil {
			return fmt.Errorf("could not auto-convert new parent page to section: %w", err)
		}
		newParent.Kind = NodeKindSection
		newParentWasConverted = true
	}

	if newParent.Kind != NodeKindSection {
		return fmt.Errorf("destination parent must be a section, got %q", newParent.Kind)
	}

	// 保存子列表、位置和节点元信息，以便后续多次存储调用失败时执行恢复尝试。
	previousOldChildren := append([]*PageNode(nil), oldParent.Children...)
	previousOldPositions := snapshotChildPositions(oldParent.Children)
	previousNewChildren := append([]*PageNode(nil), newParent.Children...)
	previousNewPositions := snapshotChildPositions(newParent.Children)
	previousPosition := node.Position
	previousMetadata := node.Metadata

	if err := t.store.MoveNode(node, newParent); err != nil {
		return fmt.Errorf("could not persist parent change: %w", err)
	}

	for i, e := range oldParent.Children {
		if e.ID == id {
			oldParent.Children = append(oldParent.Children[:i], oldParent.Children[i+1:]...)
			break
		}
	}

	// 插入前已从旧列表移除节点，同父移动时此长度也反映移除后的列表。
	insertAt := len(newParent.Children)
	if position >= 0 && position < len(newParent.Children) {
		insertAt = position
	}
	newParent.Children = append(newParent.Children, nil)
	copy(newParent.Children[insertAt+1:], newParent.Children[insertAt:])
	newParent.Children[insertAt] = node
	node.Position = insertAt
	node.Parent = newParent
	t.rebuildChildSlugIndexForParentLocked(oldParent)
	t.rebuildChildSlugIndexForParentLocked(newParent)
	node.Metadata.UpdatedAt = time.Now().UTC()
	node.Metadata.LastAuthorID = userID

	// 重新编号前先按现有 Position 稳定排序，因此最终次序还受到其他子节点位置值影响。
	t.reindexPositions(newParent)
	t.reindexPositions(oldParent)

	if err := t.store.SaveChildOrder(oldParent); err != nil {
		rollbackErr := t.rollbackMovedNodeLocked(node, oldParent, newParent, previousOldChildren, previousOldPositions, previousNewChildren, previousNewPositions, previousPosition, previousMetadata, newParentWasConverted)
		if rollbackErr != nil {
			return errors.Join(fmt.Errorf("could not persist source child order: %w", err), fmt.Errorf(errRollbackMovedNodeFailed, rollbackErr))
		}
		return fmt.Errorf("could not persist source child order: %w", err)
	}
	if newParent != oldParent {
		if err := t.store.SaveChildOrder(newParent); err != nil {
			rollbackErr := t.rollbackMovedNodeLocked(node, oldParent, newParent, previousOldChildren, previousOldPositions, previousNewChildren, previousNewPositions, previousPosition, previousMetadata, newParentWasConverted)
			if rollbackErr != nil {
				return errors.Join(fmt.Errorf("could not persist destination child order: %w", err), fmt.Errorf(errRollbackMovedNodeFailed, rollbackErr))
			}
			return fmt.Errorf("could not persist destination child order: %w", err)
		}
	}

	if err := t.store.SyncFrontmatterIfExists(node); err != nil {
		rollbackErr := t.rollbackMovedNodeLocked(node, oldParent, newParent, previousOldChildren, previousOldPositions, previousNewChildren, previousNewPositions, previousPosition, previousMetadata, newParentWasConverted)
		if rollbackErr != nil {
			return errors.Join(fmt.Errorf("could not sync moved node frontmatter: %w", err), fmt.Errorf(errRollbackMovedNodeFailed, rollbackErr))
		}
		return fmt.Errorf("could not sync moved node frontmatter: %w", err)
	}

	return nil
}

// snapshotChildPositions 按标识保存非空子节点的位置，同标识后项覆盖前项。
func snapshotChildPositions(children []*PageNode) map[string]int {
	positions := make(map[string]int, len(children))
	for _, child := range children {
		if child == nil {
			continue
		}
		positions[child.ID] = child.Position
	}
	return positions
}

// restoreChildSnapshot 复制保存的子列表，并恢复每个非空子节点的父指针和已记录位置。
// 未保存位置的节点保留当前 Position，空父节点不执行操作。
func restoreChildSnapshot(parent *PageNode, children []*PageNode, positions map[string]int) {
	if parent == nil {
		return
	}

	parent.Children = append([]*PageNode(nil), children...)
	for _, child := range parent.Children {
		if child == nil {
			continue
		}
		child.Parent = parent
		if pos, ok := positions[child.ID]; ok {
			child.Position = pos
		}
	}
}

// rollbackMovedNodeLocked 在树锁内尝试恢复父记录、内存快照、目标类型和两侧排序。
// 各恢复步骤失败会累计错误并继续处理后续步骤，不保证数据库与内存全部恢复成功。
func (t *TreeService) rollbackMovedNodeLocked(node *PageNode, oldParent *PageNode, newParent *PageNode, previousOldChildren []*PageNode, previousOldPositions map[string]int, previousNewChildren []*PageNode, previousNewPositions map[string]int, previousPosition int, previousMetadata PageMetadata, newParentWasConverted bool) error {
	var rollbackErr error

	if moveErr := t.store.MoveNode(node, oldParent); moveErr != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore parent record: %w", moveErr))
	}

	restoreChildSnapshot(oldParent, previousOldChildren, previousOldPositions)
	if newParent != oldParent {
		restoreChildSnapshot(newParent, previousNewChildren, previousNewPositions)
	}
	t.rebuildChildSlugIndexForParentLocked(oldParent)
	if newParent != oldParent {
		t.rebuildChildSlugIndexForParentLocked(newParent)
	}
	node.Parent = oldParent
	node.Position = previousPosition
	node.Metadata = previousMetadata

	if newParentWasConverted && newParent != nil && newParent.ID != "root" && len(newParent.Children) == 0 {
		if convertErr := t.store.ConvertNode(newParent, NodeKindPage); convertErr != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("convert destination parent back to page: %w", convertErr))
		} else {
			newParent.Kind = NodeKindPage
		}
	}

	if err := t.store.SaveChildOrder(oldParent); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore source child order: %w", err))
	}
	if newParent != oldParent && newParent.Kind == NodeKindSection {
		if err := t.store.SaveChildOrder(newParent); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore destination child order: %w", err))
		}
	}

	return rollbackErr
}

// SetPinned 校验版本后先保存置顶元信息，再设置内存标记并返回包含正文的页面。
// 此入口不更新节点修改时间和最后作者，也不生成新的版本值。
func (t *TreeService) SetPinned(id string, version string, pinned bool) (*Page, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tree == nil {
		return nil, ErrTreeNotLoaded
	}

	node := t.getNodeByIDLocked(id)
	if node == nil {
		return nil, ErrPageNotFound
	}

	if err := checkNodeVersion(node, version); err != nil {
		return nil, err
	}

	content, err := t.store.SetPinnedFrontmatter(node, pinned)
	if err != nil {
		return nil, fmt.Errorf("set pinned: %w", err)
	}

	node.Pinned = pinned
	return &Page{PageNode: node, Content: content}, nil
}

// SortPages 要求提供父节点全部子标识各一次，按给定顺序重排并连续编号。
// 存储失败会恢复内存子列表与位置，数据库事务行为由存储入口负责。
func (t *TreeService) SortPages(parentID string, orderedIDs []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tree == nil {
		return ErrTreeNotLoaded
	}

	parent := t.tree

	if parentID != "" && parentID != "root" {
		parent = t.getNodeByIDLocked(parentID)
		if parent == nil {
			return ErrParentNotFound
		}
	}

	// 长度必须相等，结合后面的成员和重复检查确保列表是完整子节点排列。
	if len(orderedIDs) != len(parent.Children) {
		return fmt.Errorf("number of ordered IDs does not match the number of children: %w", ErrInvalidSortOrder)
	}

	// 每个输入标识都必须属于当前父节点，不接受其他分区页面。
	existingIDs := make(map[string]bool)
	for _, child := range parent.Children {
		existingIDs[child.ID] = true
	}
	for _, id := range orderedIDs {
		if !existingIDs[id] {
			return fmt.Errorf("invalid ID in sort order, ID: %s - %w", id, ErrInvalidSortOrder)
		}
	}

	// 重复标识单独返回错误，不让重复项挤掉列表中的其他子节点。
	seen := make(map[string]bool)
	for _, id := range orderedIDs {
		if seen[id] {
			return fmt.Errorf("duplicate ID in sort order: %s", id)
		}
		seen[id] = true
	}

	previousChildren := append([]*PageNode(nil), parent.Children...)
	previousPositions := make(map[string]int, len(parent.Children))
	for _, child := range parent.Children {
		previousPositions[child.ID] = child.Position
	}

	// 输入下标就是期望位置，用映射供稳定排序比较函数查找。
	positions := make(map[string]int)
	for i, id := range orderedIDs {
		positions[id] = i
	}

	// 原地重排父节点的子列表，所有子节点都已通过完整性检查。
	sort.SliceStable(parent.Children, func(i, j int) bool {
		return positions[parent.Children[i].ID] < positions[parent.Children[j].ID]
	})

	// 列表下标同步到 Position，再一起交给存储层保存。
	for i, child := range parent.Children {
		child.Position = i
	}

	if err := t.store.SaveChildOrder(parent); err != nil {
		parent.Children = previousChildren
		for _, child := range parent.Children {
			if pos, ok := previousPositions[child.ID]; ok {
				child.Position = pos
			}
		}
		return fmt.Errorf(errPersistChildOrderFailed, err)
	}

	return nil
}

// reindexPositions 先按当前 Position 稳定排序，再将位置重写为从零开始的连续下标。
// 相同位置保留进入排序时的相对顺序，不单纯沿现有列表直接编号。
func (t *TreeService) reindexPositions(parent *PageNode) {
	sort.SliceStable(parent.Children, func(i, j int) bool {
		return parent.Children[i].Position < parent.Children[j].Position
	})
	for i, child := range parent.Children {
		child.Position = i
	}
}
