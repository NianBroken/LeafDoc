// 本文件定义标签计数与按标签筛选的页面摘要，转换过程不执行标签索引查询。
package dto

import (
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// TagCount 保存一个标签及其关联数量，具体统计口径由调用方提供。
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// TaggedPage 包含路径、摘要、标签和作者元数据，不包含正文或递归子节点。
type TaggedPage struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Path         string          `json:"path"`
	Excerpt      string          `json:"excerpt,omitempty"`
	Tags         []string        `json:"tags"`
	CreatedAt    string          `json:"createdAt,omitempty"`
	UpdatedAt    string          `json:"updatedAt,omitempty"`
	CreatorID    string          `json:"creatorId,omitempty"`
	LastAuthorID string          `json:"lastAuthorId,omitempty"`
	LastAuthor   *auth.UserLabel `json:"lastAuthor,omitempty"`
}

// ToTaggedPage 使用传入标签和摘要，不从节点重新提取它们。空标签切片转为非空空数组，
// 非空切片直接复用而不复制底层数组；调用方应避免转换后并发修改。
// 零时间不格式化，用户标签解析错误不阻止返回页面摘要。
func ToTaggedPage(node *tree.PageNode, pageTags []string, excerpt string, userResolver *auth.UserResolver) *TaggedPage {
	if pageTags == nil {
		pageTags = []string{}
	}

	p := &TaggedPage{
		ID:           node.ID,
		Title:        node.Title,
		Path:         BuildPathFromNode(node),
		Excerpt:      excerpt,
		Tags:         pageTags,
		LastAuthorID: node.Metadata.LastAuthorID,
		CreatorID:    node.Metadata.CreatorID,
	}

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
