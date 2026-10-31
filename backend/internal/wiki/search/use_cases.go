package search

import (
	"context"
	"sort"
	"strings"

	coresearch "github.com/NianBroken/LeafDoc/internal/core/search"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/shared/htmlutil"
	coretags "github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/http/dto"
)

// ErrSearchUnavailable 表示没有可用搜索索引服务，不等同于一次具体 SQL 查询失败。
var ErrSearchUnavailable = sharederrors.NewLocalizedError(
	ErrCodeSearchUnavailable,
	"Search is currently unavailable",
	"search is currently unavailable",
	nil,
)

// SearchInput 同时支持全文关键词和标签范围，并携带调用方提供的偏移与条数。
type SearchInput struct {
	Query  string
	Tags   []string
	Offset int
	Limit  int
}

// SearchOutput 包含搜索结果及其总数、分页和标签分组统计。
type SearchOutput struct {
	Result *coresearch.SearchResult
}

// SearchUseCase 先由标签确定候选页面，再执行全文或纯标签查询并附带标签信息。
// 关键词查询与全量匹配标识查询分别执行，不构成同一个数据库快照。
type SearchUseCase struct {
	index *coresearch.MySQLIndex
	tags  *coretags.TagsService
	tree  *tree.TreeService
}

// NewSearchUseCase 保存索引、标签与树依赖，允许索引为空并在执行时返回领域错误。
func NewSearchUseCase(idx *coresearch.MySQLIndex, tags *coretags.TagsService, tree *tree.TreeService) *SearchUseCase {
	return &SearchUseCase{index: idx, tags: tags, tree: tree}
}

// Execute 不使用传入上下文；索引缺失直接返回不可用错误。
// 请求有标签时即使标签服务缺失也使用非 nil 的空候选集合，保留限制为空的语义。
// 空白关键词且候选非空时走树查询，其余情况交给搜索索引处理。
func (uc *SearchUseCase) Execute(_ context.Context, in SearchInput) (*SearchOutput, error) {
	if uc.index == nil {
		return nil, ErrSearchUnavailable
	}

	var pageIDs []string
	if len(in.Tags) > 0 {
		pageIDs = []string{}
		if uc.tags != nil {
			var err error
			pageIDs, err = uc.tags.GetPageIDsByTags(normalizeTags(in.Tags))
			if err != nil {
				return nil, err
			}
		}
	}

	if strings.TrimSpace(in.Query) == "" && len(pageIDs) > 0 {
		return uc.searchByTags(pageIDs, in.Offset, in.Limit)
	}

	result, err := uc.index.Search(in.Query, pageIDs, in.Offset, in.Limit)
	if err != nil {
		return nil, err
	}

	// 分组统计使用全部命中页面而非当前页，第二次查询失败仍使整个请求失败。
	fullMatchPageIDs, err := uc.index.SearchPageIDs(in.Query, pageIDs)
	if err != nil {
		return nil, err
	}

	uc.attachTags(result.Items)
	result.TagFacets = uc.buildTagFacets(fullMatchPageIDs)
	return &SearchOutput{Result: result}, nil
}

// searchByTags 按树中的当前节点建立纯标签结果，跳过查找失败或已不存在的页面。
// 标题作 HTML 文本转义后按标题、路径排序，再截取分页并只读取显示页面的摘要。
// 标签或树服务缺失时返回空结果，此分支保留原始分页参数。
func (uc *SearchUseCase) searchByTags(pageIDs []string, offset, limit int) (*SearchOutput, error) {
	if uc.tags == nil || uc.tree == nil {
		return &SearchOutput{
			Result: &coresearch.SearchResult{
				Count:     0,
				Items:     []coresearch.SearchResultItem{},
				Offset:    offset,
				Limit:     limit,
				TagFacets: []coresearch.SearchTagFacet{},
			},
		}, nil
	}

	items := make([]coresearch.SearchResultItem, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		node, err := uc.tree.FindPageByID(pageID)
		if err != nil || node == nil {
			continue
		}

		items = append(items, coresearch.SearchResultItem{
			PageID: node.ID,
			Title:  htmlutil.EscapeText(node.Title),
			Path:   dto.BuildPathFromNode(node),
			Kind:   string(node.Kind),
			Rank:   1,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Title == items[j].Title {
			return items[i].Path < items[j].Path
		}
		return items[i].Title < items[j].Title
	})

	// 实际分页分支将负偏移归零、非正条数设为二十，超出总数的偏移压到尾部。
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}

	count := len(items)
	if offset > count {
		offset = count
	}
	end := offset + limit
	if end > count {
		end = count
	}
	pagedItems := items[offset:end]
	displayedIDs := make([]string, len(pagedItems))
	for i, item := range pagedItems {
		displayedIDs[i] = item.PageID
	}
	// 摘要读取失败是主流程错误；标签附带读取与分组统计则采用各自的空结果回退。
	excerpts, err := uc.tags.GetExcerptsForPages(displayedIDs)
	if err != nil {
		return nil, err
	}
	for i := range pagedItems {
		pagedItems[i].Excerpt = excerpts[pagedItems[i].PageID]
	}
	uc.attachTags(pagedItems)

	return &SearchOutput{
		Result: &coresearch.SearchResult{
			Count:     count,
			Items:     pagedItems,
			Offset:    offset,
			Limit:     limit,
			TagFacets: uc.buildTagFacets(pageIDs),
		},
	}, nil
}

// attachTags 批量读取非空页面标识的标签，并原地写入结果切片中的元素。
// 读取失败直接保留原结果，成功但某页无标签时明确设置为空切片。
func (uc *SearchUseCase) attachTags(items []coresearch.SearchResultItem) {
	if uc.tags == nil || len(items) == 0 {
		return
	}

	pageIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.PageID != "" {
			pageIDs = append(pageIDs, item.PageID)
		}
	}
	if len(pageIDs) == 0 {
		return
	}

	tagsByPage, err := uc.tags.GetTagsForPages(pageIDs)
	if err != nil {
		return
	}

	for i := range items {
		if tags, ok := tagsByPage[items[i].PageID]; ok {
			items[i].Tags = tags
		} else {
			items[i].Tags = []string{}
		}
	}
}

// normalizeTags 去除两端空白、转小写并按首次出现顺序去重，空标签不进入查询。
func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
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

// buildTagFacets 对输入页面列表逐项累计标签出现次数，按次数降序、同次数按标签升序排列。
// 服务缺失、空页面列表或读取失败都返回空切片，不将附带统计错误升级为查询失败。
// 此处不对 pageIDs 去重，重复页面标识会再次参与累计。
func (uc *SearchUseCase) buildTagFacets(pageIDs []string) []coresearch.SearchTagFacet {
	if uc.tags == nil || len(pageIDs) == 0 {
		return []coresearch.SearchTagFacet{}
	}

	tagsByPage, err := uc.tags.GetTagsForPages(pageIDs)
	if err != nil {
		return []coresearch.SearchTagFacet{}
	}

	counts := make(map[string]int)
	for _, pageID := range pageIDs {
		for _, tag := range tagsByPage[pageID] {
			counts[tag]++
		}
	}

	facets := make([]coresearch.SearchTagFacet, 0, len(counts))
	for tag, count := range counts {
		facets = append(facets, coresearch.SearchTagFacet{
			Tag:   tag,
			Count: count,
		})
	}

	sort.Slice(facets, func(i, j int) bool {
		if facets[i].Count == facets[j].Count {
			return facets[i].Tag < facets[j].Tag
		}
		return facets[i].Count > facets[j].Count
	})

	return facets
}

// GetIndexingStatusOutput 返回状态副本，未配置状态对象时 Status 为空。
type GetIndexingStatusOutput struct {
	Status *coresearch.IndexingStatus
}

// GetIndexingStatusUseCase 只读取后台任务的共享状态，不触发索引执行。
type GetIndexingStatusUseCase struct {
	status *coresearch.IndexingStatus
}

// NewGetIndexingStatusUseCase 保存可选状态对象。
func NewGetIndexingStatusUseCase(s *coresearch.IndexingStatus) *GetIndexingStatusUseCase {
	return &GetIndexingStatusUseCase{status: s}
}

// Execute 使用状态对象的同步快照入口，避免将内部可变对象直接交给响应编码。
func (uc *GetIndexingStatusUseCase) Execute(_ context.Context) *GetIndexingStatusOutput {
	if uc.status == nil {
		return &GetIndexingStatusOutput{Status: nil}
	}
	return &GetIndexingStatusOutput{Status: uc.status.Snapshot()}
}
