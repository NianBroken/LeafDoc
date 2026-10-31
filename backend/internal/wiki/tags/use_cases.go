package tags

import (
	"context"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	coretags "github.com/NianBroken/LeafDoc/internal/core/tags"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/http/dto"
)

// GetTagsInput 同时提供文字筛选、当前选择及候选条数限制。
type GetTagsInput struct {
	Filter   string
	Selected []string
	Limit    int
}

// GetTagsOutput 保存标签与对应页面数量，空结果统一使用空切片。
type GetTagsOutput struct {
	Tags []coretags.TagCount
}

// GetTagsUseCase 将候选查询交给标签服务，有已选标签时使用选择范围对应的查询入口。
type GetTagsUseCase struct {
	svc *coretags.TagsService
}

// NewGetTagsUseCase 保存已有标签服务，不建立独立索引。
func NewGetTagsUseCase(svc *coretags.TagsService) *GetTagsUseCase {
	return &GetTagsUseCase{svc: svc}
}

// Execute 将非正条数设为五十、超过二百的条数压到二百，并规范化筛选及选择。
// 服务错误直接返回，不将失败当作空候选；上下文未用于取消查询。
func (uc *GetTagsUseCase) Execute(_ context.Context, in GetTagsInput) (*GetTagsOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	filter := strings.ToLower(strings.TrimSpace(in.Filter))
	selected := normalizeTags(in.Selected)

	var (
		tags []coretags.TagCount
		err  error
	)
	if len(selected) == 0 {
		tags, err = uc.svc.GetAllTags(filter, limit)
	} else {
		tags, err = uc.svc.GetAllTagsForSelection(filter, selected, limit)
	}
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []coretags.TagCount{}
	}
	return &GetTagsOutput{Tags: tags}, nil
}

// GetPagesByTagsInput 提供标签列表，由用例统一去空、转小写并去重。
type GetPagesByTagsInput struct {
	Tags []string
}

// GetPagesByTagsOutput 包含当前仍在树中的页面及其标签、摘要与作者信息。
type GetPagesByTagsOutput struct {
	Pages []*dto.TaggedPage
}

// GetPagesByTagsUseCase 先取得命中标识，再批量读取标签及摘要，并逐页解析树节点。
type GetPagesByTagsUseCase struct {
	svc          *coretags.TagsService
	treeService  *tree.TreeService
	userResolver *auth.UserResolver
}

// NewGetPagesByTagsUseCase 保存索引、树与作者解析器，供结果转换共同使用。
func NewGetPagesByTagsUseCase(svc *coretags.TagsService, treeService *tree.TreeService, userResolver *auth.UserResolver) *GetPagesByTagsUseCase {
	return &GetPagesByTagsUseCase{svc: svc, treeService: treeService, userResolver: userResolver}
}

// Execute 对空选择或无匹配页面返回空切片，批量索引读取失败则返回错误。
// 单页树查找失败或节点为空时跳过，不让已经失效的索引标识中断其他结果。
// 结果保持标识列表的遍历顺序，不额外排序；传入上下文不参与执行。
func (uc *GetPagesByTagsUseCase) Execute(_ context.Context, in GetPagesByTagsInput) (*GetPagesByTagsOutput, error) {
	normalized := normalizeTags(in.Tags)
	if len(normalized) == 0 {
		return &GetPagesByTagsOutput{Pages: []*dto.TaggedPage{}}, nil
	}

	pageIDs, err := uc.svc.GetPageIDsByTags(normalized)
	if err != nil {
		return nil, err
	}
	if len(pageIDs) == 0 {
		return &GetPagesByTagsOutput{Pages: []*dto.TaggedPage{}}, nil
	}

	tagsPerPage, err := uc.svc.GetTagsForPages(pageIDs)
	if err != nil {
		return nil, err
	}

	excerptsPerPage, err := uc.svc.GetExcerptsForPages(pageIDs)
	if err != nil {
		return nil, err
	}

	pages := make([]*dto.TaggedPage, 0, len(pageIDs))
	for _, id := range pageIDs {
		node, err := uc.treeService.FindPageByID(id)
		if err != nil || node == nil {
			continue
		}
		pages = append(pages, dto.ToTaggedPage(node, tagsPerPage[id], excerptsPerPage[id], uc.userResolver))
	}

	return &GetPagesByTagsOutput{Pages: pages}, nil
}

// normalizeTags 按首次出现顺序保留去空白并转小写后的非空唯一标签。
func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		n := strings.ToLower(strings.TrimSpace(t))
		if n == "" {
			continue
		}
		if _, exists := seen[n]; exists {
			continue
		}
		seen[n] = struct{}{}
		result = append(result, n)
	}
	return result
}
