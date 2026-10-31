// 本文件定义页面树节点，并提供层级路径、版本标识和稳定结构哈希。
package tree

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"sort"
	"time"
)

// PageMetadata 保存页面创建与更新时间，以及创建者和最近作者 ID。
type PageMetadata struct {
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	CreatorID    string    `json:"creatorId"`
	LastAuthorID string    `json:"lastAuthorId"`
}

// NodeKind 使用 page 和 section 两种传输值表示页面与章节节点。
type NodeKind string

const (
	NodeKindPage    NodeKind = "page"
	NodeKindSection NodeKind = "section"
)

// PageNode 保存树中一个节点的身份、显示信息、父子引用和元信息，不直接保存正文。
// Parent 不参与 JSON 序列化，避免沿双向父子关系形成循环结构。
type PageNode struct {
	ID       string      `json:"id"`       // 节点稳定标识，与标题和位置分离。
	Title    string      `json:"title"`    // 面向阅读者的节点标题。
	Slug     string      `json:"slug"`     // 当前层路径片段，不是完整层级路径。
	Children []*PageNode `json:"children"` // 直接子节点引用，完整子树由这些引用继续展开。
	Position int         `json:"position"` // 同级顺序字段，哈希排序时优先使用。
	Parent   *PageNode   `json:"-"`
	Pinned   bool        `json:"pinned,omitempty"` // 侧栏固定状态，为 false 时省略 JSON 字段。

	Kind     NodeKind     `json:"kind"`     // 页面或章节类别。
	Metadata PageMetadata `json:"metadata"` // 创建、更新与作者元信息。
}

// HasChildren 仅检查子节点切片长度，不检查条目是否为 nil。
func (p *PageNode) HasChildren() bool {
	return len(p.Children) > 0
}

// IsChildOf 按参数 ID 检查本节点的子级，而不是检查本节点是否属于某个父级。
// recursive 为 true 时深度优先继续搜索后代，调用方须提供无环且子节点非 nil 的树。
func (p *PageNode) IsChildOf(childID string, recursive bool) bool {
	for _, child := range p.Children {
		if child.ID == childID {
			return true
		}
		if recursive && child.IsChildOf(childID, recursive) {
			return true
		}
	}
	return false
}

// CalculatePath 递归连接祖先和当前 slug，根节点 slug 为空或 root 时返回空串。
// 普通根节点保留自身 slug，子节点始终在父路径后拼接斜杠和当前 slug。
func (p *PageNode) CalculatePath() string {
	if p.Parent == nil {
		if p.Slug == "" || p.Slug == "root" {
			return ""
		}
		return p.Slug
	}
	return p.Parent.CalculatePath() + "/" + p.Slug
}

// Version 将非零更新时间转为 UTC RFC3339Nano 字符串，供乐观版本比较。
// nil 节点或零更新时间返回空版本，不通过正文或其他字段计算版本值。
func (p *PageNode) Version() string {
	if p == nil {
		return ""
	}
	if p.Metadata.UpdatedAt.IsZero() {
		return ""
	}
	return p.Metadata.UpdatedAt.UTC().Format(time.RFC3339Nano)
}

// Hash 计算本节点及全部后代的 SHA-256，包含元信息但不沿 Parent 引用遍历。
// 不修改原子节点次序，哈希使用排序后的副本；正文不在 PageNode 中因此不参与。
func (p *PageNode) Hash() string {
	sum := p.hashSum(true) // 包含节点元信息。
	return hex.EncodeToString(sum[:])
}

// hashSum 把节点载荷直接写入摘要器，避免先构造完整树序列化字符串。
func (p *PageNode) hashSum(includeMetadata bool) [32]byte {
	h := sha256.New()

	// 载荷按固定字段顺序和深度优先次序写入，输出为固定三十二字节摘要。
	p.writeHashPayload(h, includeMetadata)

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// writeHashPayload 用字段标签、长度前缀和值区分不同载荷边界，不写父节点。
// 这些内部写入函数忽略 Writer 错误，当前哈希路径使用不会返回写错误的摘要器。
func (p *PageNode) writeHashPayload(w io.Writer, includeMetadata bool) {
	// 字段名称也进入摘要，防止相邻值组合混淆。
	writeString(w, "id")
	writeString(w, p.ID)
	writeString(w, "title")
	writeString(w, p.Title)
	writeString(w, "slug")
	writeString(w, p.Slug)
	writeString(w, "kind")
	writeString(w, string(p.Kind))
	writeString(w, "position")
	writeInt64(w, int64(p.Position))

	writeString(w, "pinned")
	if p.Pinned {
		writeString(w, "true")
	} else {
		writeString(w, "false")
	}

	if includeMetadata {
		writeString(w, "meta.createdAt")
		writeTime(w, p.Metadata.CreatedAt)
		writeString(w, "meta.updatedAt")
		writeTime(w, p.Metadata.UpdatedAt)
		writeString(w, "meta.creatorId")
		writeString(w, p.Metadata.CreatorID)
		writeString(w, "meta.lastAuthorId")
		writeString(w, p.Metadata.LastAuthorID)
	}

	// 复制子节点后排序，非空节点按位置、相同位置按 ID 排列。
	children := make([]*PageNode, 0, len(p.Children))
	children = append(children, p.Children...)

	sort.SliceStable(children, func(i, j int) bool {
		if children[i] == nil || children[j] == nil {
			return children[j] != nil // 一个为空时，nil 排在非空节点之前。
		}
		if children[i].Position != children[j].Position {
			return children[i].Position < children[j].Position
		}
		return children[i].ID < children[j].ID
	})

	writeString(w, "children.count")
	writeInt64(w, int64(len(children)))

	for _, ch := range children {
		if ch == nil {
			writeString(w, "child.nil")
			continue
		}
		// 子树用开始和结束标签包围，nil 条目使用独立标签而不递归。
		writeString(w, "child.begin")
		ch.writeHashPayload(w, includeMetadata)
		writeString(w, "child.end")
	}
}

// writeString 写入大端 uint32 字节长度，再写原字符串字节，不按字符数计长。
func writeString(w io.Writer, s string) {
	_ = binary.Write(w, binary.BigEndian, uint32(len(s)))
	_, _ = io.WriteString(w, s)
}

// writeInt64 用固定大端八字节表示整数，避免文本格式差异影响摘要。
func writeInt64(w io.Writer, v int64) {
	_ = binary.Write(w, binary.BigEndian, v)
}

// writeTime 将零时间写为零，其余时间写为 UTC Unix 纳秒。
func writeTime(w io.Writer, t time.Time) {
	if t.IsZero() {
		writeInt64(w, 0)
		return
	}
	writeInt64(w, t.UTC().UnixNano())
}
