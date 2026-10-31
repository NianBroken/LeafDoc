// 本文件维护节点 ID、标题及父级子 slug 索引，带 Locked 后缀的方法由调用方持锁。
package tree

import (
	"strings"
)

// FindPagesByTitle 裁去查询标题边缘空白并转小写后读取标题索引，未加载或无结果返回 nil。
// 返回的是索引切片本身，调用方不应修改；整体重建按前序遍历登记，增量登记按追加次序保存。
func (t *TreeService) FindPagesByTitle(title string) []*PageNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tree == nil {
		return nil
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}

	nodes := t.nodesByTitle[strings.ToLower(title)]
	if len(nodes) == 0 {
		return nil
	}
	return nodes
}

// FindPageByID 持读锁取得结构节点，不加载正文，返回节点仍属于当前内存树。
func (t *TreeService) FindPageByID(id string) (*PageNode, error) {
	var result *PageNode
	err := t.withRLockedTree(func() error {
		if t.tree == nil {
			return ErrTreeNotLoaded
		}

		result = t.getNodeByIDLocked(id)
		if result == nil {
			return ErrPageNotFound
		}

		return nil
	})

	return result, err
}

// getNodeByIDLocked 把 root 特殊映射到根节点，空 ID 返回 nil，其余直接查 ID 索引。
func (t *TreeService) getNodeByIDLocked(id string) *PageNode {
	if id == "" {
		return nil
	}
	if id == "root" {
		return t.tree
	}

	return t.nodesByID[id]
}

// rebuildIndexesLocked 丢弃旧索引，按当前 Children 次序深度优先重新登记。
// nil 节点跳过，空 ID 和 root 不进入 ID 与标题索引，所有有效节点都建立子 slug 索引。
func (t *TreeService) rebuildIndexesLocked() {
	t.nodesByID = make(map[string]*PageNode)
	t.nodesByTitle = make(map[string][]*PageNode)
	t.childSlugs = make(map[string]map[string]*PageNode)

	if t.tree == nil {
		return
	}

	var walk func(node *PageNode)
	walk = func(node *PageNode) {
		if node == nil {
			return
		}

		if node.ID != "" && node.ID != "root" {
			t.nodesByID[node.ID] = node
			t.addTitleIndexForNodeLocked(node)
		}
		t.rebuildChildSlugIndexForParentLocked(node)

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(t.tree)
}

// rebuildChildSlugIndexForParentLocked 为父节点重建小写 slug 映射，重复键由后项覆盖。
func (t *TreeService) rebuildChildSlugIndexForParentLocked(parent *PageNode) {
	if parent == nil {
		return
	}

	index := make(map[string]*PageNode, len(parent.Children))
	for _, child := range parent.Children {
		if child == nil {
			continue
		}
		index[strings.ToLower(child.Slug)] = child
	}
	t.childSlugs[parent.ID] = index
}

// addTitleIndexForNodeLocked 将非空标题节点追加到小写标题组，不去重也不裁标题空白。
func (t *TreeService) addTitleIndexForNodeLocked(node *PageNode) {
	if node == nil || node.Title == "" {
		return
	}
	key := strings.ToLower(node.Title)
	t.nodesByTitle[key] = append(t.nodesByTitle[key], node)
}

// removeTitleIndexForNodeLocked 在对应标题组原切片上过滤相同 ID，空组删除映射键。
func (t *TreeService) removeTitleIndexForNodeLocked(node *PageNode) {
	if node == nil || node.Title == "" {
		return
	}
	key := strings.ToLower(node.Title)
	nodes := t.nodesByTitle[key]
	filtered := nodes[:0]
	for _, n := range nodes {
		if n.ID != node.ID {
			filtered = append(filtered, n)
		}
	}
	if len(filtered) == 0 {
		delete(t.nodesByTitle, key)
	} else {
		t.nodesByTitle[key] = filtered
	}
}

// indexNodeLocked 登记单节点，并重建该节点和其父级的子 slug 索引，不递归登记后代。
func (t *TreeService) indexNodeLocked(node *PageNode) {
	if node == nil {
		return
	}

	if node.ID != "" && node.ID != "root" {
		t.nodesByID[node.ID] = node
		t.addTitleIndexForNodeLocked(node)
	}
	t.rebuildChildSlugIndexForParentLocked(node)
	if node.Parent != nil {
		t.rebuildChildSlugIndexForParentLocked(node.Parent)
	}
}

// removeNodeIndexLocked 递归移除整棵子树的三类索引，结束后重建原父级子 slug 映射。
// 本方法不修改 Children 切片，调用方须先让父子结构与预期删除结果一致。
func (t *TreeService) removeNodeIndexLocked(node *PageNode) {
	if node == nil {
		return
	}

	var walk func(current *PageNode)
	walk = func(current *PageNode) {
		if current == nil {
			return
		}

		delete(t.nodesByID, current.ID)
		delete(t.childSlugs, current.ID)
		t.removeTitleIndexForNodeLocked(current)

		for _, child := range current.Children {
			walk(child)
		}
	}

	walk(node)
	if node.Parent != nil {
		t.rebuildChildSlugIndexForParentLocked(node.Parent)
	}
}

// findChildBySlugInParentLocked 优先用小写索引查找，只有父级没有索引时才线性回退。
// 索引存在但未命中不会扫描切片，父节点为空直接返回 nil。
func (t *TreeService) findChildBySlugInParentLocked(parent *PageNode, slug string) *PageNode {
	if parent == nil {
		return nil
	}

	if index, ok := t.childSlugs[parent.ID]; ok {
		return index[strings.ToLower(slug)]
	}

	for _, child := range parent.Children {
		if child != nil && strings.EqualFold(child.Slug, slug) {
			return child
		}
	}

	return nil
}

// findChildBySlugExactInParentLocked 逐项精确比较 slug，不使用忽略大小写的索引。
func (t *TreeService) findChildBySlugExactInParentLocked(parent *PageNode, slug string) *PageNode {
	if parent == nil {
		return nil
	}

	for _, child := range parent.Children {
		if child != nil && child.Slug == slug {
			return child
		}
	}

	return nil
}
