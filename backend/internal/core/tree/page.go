// 本文件把树节点与页面正文组合，并定义永久链接解析所需的目标信息。
package tree

// Page 嵌入节点引用，Content 为正文，RawContent 保留含前置字段的原始内容。
// 节点不是深复制，修改嵌入节点会作用于该引用指向的对象。
type Page struct {
	*PageNode
	Content    string `json:"content"`
	RawContent string `json:"-"` // 含前置字段的完整原始内容，不参与 JSON 序列化。
}

// PermalinkTarget 向永久链接入口提供稳定 ID、当前 slug 和完整路径。
type PermalinkTarget struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Path string `json:"path"`
}

// Version 直接使用嵌入节点的更新时间版本，页面或节点为空时返回空串。
func (p *Page) Version() string {
	if p == nil || p.PageNode == nil {
		return ""
	}
	return p.PageNode.Version()
}
