// 本文件转换属性筛选接口的页面摘要，不读取正文或执行属性查询。
package dto

import (
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	coreprop "github.com/NianBroken/LeafDoc/internal/core/properties"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// PropertyEntry 原样表示核心属性的值和类型，转换时不在 DTO 层强制改写类型。
type PropertyEntry struct {
	Value string `json:"value"`
	Type  string `json:"type"` // 类型直接沿用核心属性条目的 Type 字段。
}

// PropertyKeyCount 组合属性键与计数，由属性列表入口填入。
type PropertyKeyCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// PropertyPage 返回属性筛选所需页面身份、属性和作者时间摘要，不包含正文与子节点。
type PropertyPage struct {
	ID           string                   `json:"id"`
	Title        string                   `json:"title"`
	Path         string                   `json:"path"`
	Properties   map[string]PropertyEntry `json:"properties"`
	CreatedAt    string                   `json:"createdAt,omitempty"`
	UpdatedAt    string                   `json:"updatedAt,omitempty"`
	CreatorID    string                   `json:"creatorId,omitempty"`
	LastAuthorID string                   `json:"lastAuthorId,omitempty"`
	LastAuthor   *auth.UserLabel          `json:"lastAuthor,omitempty"`
}

// ToPropertyPage 将每个核心属性复制到新映射，并从节点生成完整路径。
// 零时间保留空文本并由 JSON 省略，作者标签解析失败被忽略；节点须非空。
func ToPropertyPage(node *tree.PageNode, props map[string]coreprop.PropertyEntry, userResolver *auth.UserResolver) *PropertyPage {
	apiProps := make(map[string]PropertyEntry, len(props))
	for k, e := range props {
		apiProps[k] = PropertyEntry{Value: e.Value, Type: e.Type}
	}

	p := &PropertyPage{
		ID:           node.ID,
		Title:        node.Title,
		Path:         BuildPathFromNode(node),
		Properties:   apiProps,
		CreatorID:    node.Metadata.CreatorID,
		LastAuthorID: node.Metadata.LastAuthorID,
	}

	// 与普通节点 DTO 不同，此处只有非零时间才格式化，避免输出零时间日期。
	if !node.Metadata.CreatedAt.IsZero() {
		p.CreatedAt = node.Metadata.CreatedAt.Format(time.RFC3339)
	}
	if !node.Metadata.UpdatedAt.IsZero() {
		p.UpdatedAt = node.Metadata.UpdatedAt.Format(time.RFC3339)
	}

	if userResolver != nil {
		p.LastAuthor, _ = userResolver.ResolveUserLabel(node.Metadata.LastAuthorID)
	}

	return p
}
