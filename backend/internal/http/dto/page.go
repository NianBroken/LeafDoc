// dto 定义业务路由共用的 HTTP 返回类型和转换函数。
// 本包依赖核心类型，不导入 wiki 业务装配层，避免路由与装配相互依赖。
package dto

import (
	"strings"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// NodeMetadata 保留作者标识和格式化时间，可选展示标签由用户解析器取得。
// 标签为空时省略，标识字段和时间文本仍正常返回。
type NodeMetadata struct {
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	CreatorID    string `json:"creatorId"`
	LastAuthorID string `json:"lastAuthorId"`

	Creator    *auth.UserLabel `json:"creator,omitempty"`
	LastAuthor *auth.UserLabel `json:"lastAuthor,omitempty"`
}

// Node 是页面树的 HTTP 表示，包含排序、版本、节点类型和递归子节点。
// 未置顶时 pinned 被省略，没有子节点时 children 沿用空指针切片的编码结果。
type Node struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Slug     string        `json:"slug"`
	Path     string        `json:"path"`
	Version  string        `json:"version"`
	Position int           `json:"position"`
	Kind     tree.NodeKind `json:"kind"`
	Children []*Node       `json:"children"`
	Metadata NodeMetadata  `json:"metadata"`
	Pinned   bool          `json:"pinned,omitempty"`
}

// Page 在节点字段之外提供正文、完整路径及可由调用方填充的标签和属性。
// 自身 Path 覆盖嵌入 Node 的同名 JSON 字段，避免单页入口只展示局部节点路径。
type Page struct {
	*Node
	Content    string            `json:"content"`
	Path       string            `json:"path"`
	Tags       []string          `json:"tags"`
	Properties map[string]string `json:"properties"`
}

// ToAPIPage 递归转换节点并保存正文，完整路径通过父引用生成。
// 标签和属性初始化为非空容器，不从页面前置字段或相关索引中自动加载。
func ToAPIPage(p *tree.Page, userResolver *auth.UserResolver) *Page {
	return &Page{
		Node:       ToAPINode(p.PageNode, "", userResolver),
		Content:    p.Content,
		Path:       BuildPathFromNode(p.PageNode),
		Tags:       []string{},
		Properties: map[string]string{},
	}
}

// ToAPIPageWithDepth 与普通页面转换相同，但在完整节点树转换后裁剪子节点深度。
// 深度限制不减少前面的递归转换与用户标签查询工作量。
func ToAPIPageWithDepth(p *tree.Page, userResolver *auth.UserResolver, depth int) *Page {
	return &Page{
		Node:       ToAPINodeWithDepth(p.PageNode, "", userResolver, depth),
		Content:    p.Content,
		Path:       BuildPathFromNode(p.PageNode),
		Tags:       []string{},
		Properties: map[string]string{},
	}
}

// BuildPathFromNode 沿父引用向上收集路径段，遇到 Slug 为 root 的节点即停止。
// 返回不带首尾斜杠的拼接文本，空节点返回空串，不检查父引用是否形成环。
func BuildPathFromNode(node *tree.PageNode) string {
	var parts []string
	current := node
	for current != nil && current.Slug != "root" {
		parts = append([]string{current.Slug}, parts...)
		current = current.Parent
	}
	return strings.Join(parts, "/")
}

// ToAPINode 从给定 parentPath 开始递归拼接节点路径，不自动沿父引用读取完整祖先路径。
// root 路径为空，元数据时间均按 RFC3339 格式化，包括零时间。
// 用户标签解析失败被忽略，节点和作者 ID 仍保留；调用方提供非空且无环的节点树。
func ToAPINode(node *tree.PageNode, parentPath string, userResolver *auth.UserResolver) *Node {
	path := node.Slug
	if node.Slug == "root" {
		path = ""
	}
	if node.Slug != "root" && parentPath != "" {
		path = parentPath + "/" + node.Slug
	}

	// 可选解析器只影响展示标签，两个作者标识分别查询，不因相同标识在此去重。
	var creator, lastAuthor *auth.UserLabel
	if userResolver != nil {
		creator, _ = userResolver.ResolveUserLabel(node.Metadata.CreatorID)
		lastAuthor, _ = userResolver.ResolveUserLabel(node.Metadata.LastAuthorID)
	}

	apiNode := &Node{
		ID:       node.ID,
		Title:    node.Title,
		Slug:     node.Slug,
		Path:     path,
		Version:  node.Version(),
		Position: node.Position,
		Kind:     node.Kind,
		Pinned:   node.Pinned,
		Metadata: NodeMetadata{
			CreatedAt:    node.Metadata.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    node.Metadata.UpdatedAt.Format(time.RFC3339),
			CreatorID:    node.Metadata.CreatorID,
			LastAuthorID: node.Metadata.LastAuthorID,
			Creator:      creator,
			LastAuthor:   lastAuthor,
		},
	}

	// 子节点按已有切片顺序转换，不在 DTO 层重新排序。
	for _, child := range node.Children {
		apiNode.Children = append(apiNode.Children, ToAPINode(child, path, userResolver))
	}

	return apiNode
}

// pruneNodeDepth 原地裁剪已经生成的 DTO。零深度删除当前节点全部子节点，
// 负深度不裁剪，正深度逐层递减；空指针直接跳过。
func pruneNodeDepth(n *Node, depth int) {
	if n == nil {
		return
	}
	if depth == 0 {
		n.Children = nil
		return
	}
	if depth < 0 {
		return
	}
	for _, child := range n.Children {
		pruneNodeDepth(child, depth-1)
	}
}

// ToAPINodeWithDepth 先递归构造完整树，再按深度裁剪，不改变原始页面节点。
func ToAPINodeWithDepth(node *tree.PageNode, parentPath string, userResolver *auth.UserResolver, depth int) *Node {
	apiNode := ToAPINode(node, parentPath, userResolver)
	if depth < 0 {
		return apiNode
	}
	pruneNodeDepth(apiNode, depth)
	return apiNode
}
