// Package tags 提取页面标签和摘要，并通过 MySQL 索引提供筛选与批量读取。
package tags

import (
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/markdown"
)

// TagsService 连接正文提取规则与索引存储，不持有页面正文的副本。
type TagsService struct {
	store *TagsStore
}

// NewTagsService 绑定已有存储，不在构造时扫描或重建页面索引。
func NewTagsService(store *TagsStore) *TagsService {
	return &TagsService{store: store}
}

// Close 关闭存储所持连接，返回底层关闭结果。
func (s *TagsService) Close() error {
	return s.store.Close()
}

// ClearIndex 同时清空标签和摘要索引，页面内容保持原样。
func (s *TagsService) ClearIndex() error {
	return s.store.Clear()
}

// IndexPageContent 分别从同一份正文提取标签和摘要，再交给存储层一起写入。
// 标签为空时仍替换原标签集合，避免正文已经删除标签而索引继续保留旧值。
func (s *TagsService) IndexPageContent(pageID, rawContent string) error {
	tags := ExtractTagsFromContent(rawContent)
	excerpt := ExtractExcerptFromContent(rawContent)
	return s.store.SetPageIndex(pageID, tags, excerpt)
}

// SetTagsForPage 原样转交标签集合，仅替换标签，不更新摘要，也不执行标准化。
func (s *TagsService) SetTagsForPage(pageID string, tags []string) error {
	return s.store.SetTagsForPage(pageID, tags)
}

// DeleteTagsForPage 仅删除页面标签，摘要记录由其他清理方法处理。
func (s *TagsService) DeleteTagsForPage(pageID string) error {
	return s.store.DeleteTagsForPage(pageID)
}

// DeletePageIndex 同时删除单页标签和摘要，不删除页面正文。
func (s *TagsService) DeletePageIndex(pageID string) error {
	return s.store.DeletePageIndex(pageID)
}

// DeletePageIndexes 在同一存储操作中清理一组页面的标签和摘要。
func (s *TagsService) DeletePageIndexes(pageIDs []string) error {
	return s.store.DeletePageIndexes(pageIDs)
}

// GetAllTags 按前缀读取标签及关联页面数量，非正数限制表示不限制结果数量。
func (s *TagsService) GetAllTags(filter string, limit int) ([]TagCount, error) {
	return s.store.GetAllTags(filter, limit)
}

// GetAllTagsForSelection 在满足全部已选标签的页面范围内统计候选标签，并排除已选标签。
// 此转交方法不去重 selected，调用方需按存储层交集计数要求提供无重复集合。
func (s *TagsService) GetAllTagsForSelection(filter string, selected []string, limit int) ([]TagCount, error) {
	return s.store.GetAllTagsForSelection(filter, selected, limit)
}

// GetPageIDsByTags 查询同时拥有全部传入标签的页面，此处不标准化或去重输入。
func (s *TagsService) GetPageIDsByTags(tags []string) ([]string, error) {
	return s.store.GetPageIDsByTags(tags)
}

// GetTagsForPages 批量读取页面标签，无标签页面不在结果映射中单独占位。
func (s *TagsService) GetTagsForPages(pageIDs []string) (map[string][]string, error) {
	return s.store.GetTagsForPages(pageIDs)
}

// GetExcerptsForPages 批量读取摘要，不在读取过程中重新解析页面正文。
func (s *TagsService) GetExcerptsForPages(pageIDs []string) (map[string]string, error) {
	return s.store.GetExcerptsForPages(pageIDs)
}

// ExtractTagsFromContent 解析前置字段，寻找忽略大小写且去除边缘空白后等于 tags 的键。
// 找到后立即标准化其值；解析失败、无前置字段或无匹配键均返回 nil。
// 同时存在多个大小写不同的匹配键时，使用映射遍历遇到的首个键，不设固定优先级。
func ExtractTagsFromContent(content string) []string {
	fm, _, has, err := markdown.ParseFrontmatter(content)
	if err != nil || !has {
		return nil
	}

	for key, value := range fm.ExtraFields {
		if strings.EqualFold(strings.TrimSpace(key), "tags") {
			return normalizeTags(value)
		}
	}
	return nil
}

// normalizeTags 只接收解析器产生的通用列表，并只保留列表中的字符串。
// 每项去除边缘空白、转为小写、丢弃空串并去重，结果保留首次出现的次序。
func normalizeTags(value interface{}) []string {
	list, ok := value.([]interface{})
	if !ok {
		return nil
	}

	seen := make(map[string]struct{})
	result := make([]string, 0, len(list))
	for _, item := range list {
		tag, ok := item.(string)
		if !ok {
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

// WithExecutor 创建与页面事务共用执行器的服务视图，外层工作单元负责提交。
func (s *TagsService) WithExecutor(executor database.Executor) *TagsService {
	return &TagsService{store: s.store.WithExecutor(executor)}
}
