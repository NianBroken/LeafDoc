// 本文件把标签索引使用的摘要提取委托给公共 excerpt 包。
package tags

import "github.com/NianBroken/LeafDoc/internal/core/excerpt"

// ExtractExcerptFromContent 从包含前置字段的原始 Markdown 提取短摘要。
// 标签服务与其他调用方共享同一提取实现，此处不再单独定义截断或清理规则。
func ExtractExcerptFromContent(raw string) string {
	return excerpt.FromContent(raw)
}
