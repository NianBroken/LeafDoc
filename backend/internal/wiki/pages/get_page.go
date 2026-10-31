package pages

import (
	"context"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// PageQueries 汇集页面树的只读查询入口，查询结果和错误沿用树服务。
// 此对象不执行用户权限判断，访问控制由调用这些入口的上层流程负责。
type PageQueries struct {
	tree *tree.TreeService
}

// NewPageQueries 保存树服务引用，不加载树或触发数据库读取。
func NewPageQueries(t *tree.TreeService) *PageQueries {
	return &PageQueries{tree: t}
}

// GetPage 按页面标识读取页面及正文，不把标识解释为页面路径。
func (q *PageQueries) GetPage(id string) (*tree.Page, error) {
	return q.tree.GetPage(id)
}

// FindByPath 按页面路由路径查找正文，路径段的解析和错误由树服务处理。
func (q *PageQueries) FindByPath(routePath string) (*tree.Page, error) {
	return q.tree.FindPageByRoutePath(routePath)
}

// LookupPath 返回逐段路径解析结果，缺失段也保留在结果中供规划流程使用。
func (q *PageQueries) LookupPath(path string) (*tree.PathLookup, error) {
	return q.tree.LookupPagePath(path)
}

// ResolvePermalink 将稳定页面标识解析到当前路由，页面移动后仍以标识定位。
func (q *PageQueries) ResolvePermalink(id string) (*tree.PermalinkTarget, error) {
	return q.tree.ResolvePermalinkTarget(id)
}

// WikiLinkMatch 是按标题查找时返回的页面摘要，包含当前路由而不包含正文。
type WikiLinkMatch struct {
	ID    string        `json:"id"`
	Title string        `json:"title"`
	Path  string        `json:"path"`
	Kind  tree.NodeKind `json:"kind"`
}

// FindByTitleOutput 同时提供匹配列表和实际列表长度，空匹配返回空列表。
type FindByTitleOutput struct {
	Matches []WikiLinkMatch `json:"matches"`
	Count   int             `json:"count"`
}

// FindByTitle 通过树服务的不区分大小写标题索引查找节点，并逐个计算当前路由。
// 匹配顺序沿用树服务返回顺序，不在这里排序或去重。
func (q *PageQueries) FindByTitle(title string) *FindByTitleOutput {
	nodes := q.tree.FindPagesByTitle(title)
	matches := make([]WikiLinkMatch, 0, len(nodes))
	for _, n := range nodes {
		matches = append(matches, WikiLinkMatch{
			ID:    n.ID,
			Title: n.Title,
			Path:  n.CalculatePath(),
			Kind:  n.Kind,
		})
	}
	return &FindByTitleOutput{Matches: matches, Count: len(matches)}
}

// SortPagesInput 指定父节点和完整子节点标识顺序，校验由树服务实施。
type SortPagesInput struct {
	ParentID   string
	OrderedIDs []string
}

// SortPagesUseCase 将页面排序请求转交给树服务的统一排序入口。
type SortPagesUseCase struct {
	tree *tree.TreeService
}

// NewSortPagesUseCase 保存树服务依赖，不立即读取或改变页面顺序。
func NewSortPagesUseCase(t *tree.TreeService) *SortPagesUseCase {
	return &SortPagesUseCase{tree: t}
}

// Execute 提交父节点和标识列表，直接返回排序或持久化错误。
// 上下文参数未传入树服务，此入口不单独检查取消状态。
func (uc *SortPagesUseCase) Execute(_ context.Context, in SortPagesInput) error {
	return uc.tree.SortPages(in.ParentID, in.OrderedIDs)
}

// SuggestSlugInput 提供建议名称所需的父标识、当前页面标识和标题。
// 当前页面标识用于排除自身冲突，空值表示不排除任何已有页面。
type SuggestSlugInput struct {
	ParentID  string
	CurrentID string
	Title     string
}

// SuggestSlugOutput 保存生成的路径建议，无法形成有效基础名称时 Slug 为空。
type SuggestSlugOutput struct {
	Slug string
}

// SuggestSlugUseCase 读取目标父节点，并通过集中名称服务生成同级不冲突的建议。
// 建议本身不占用名称，也不建立页面，后续写入仍需检查实际冲突。
type SuggestSlugUseCase struct {
	tree *tree.TreeService
	slug *tree.SlugService
}

// NewSuggestSlugUseCase 绑定树与名称生成器，保持查询和名称规则的职责分离。
func NewSuggestSlugUseCase(t *tree.TreeService, s *tree.SlugService) *SuggestSlugUseCase {
	return &SuggestSlugUseCase{tree: t, slug: s}
}

// Execute 对空父标识或 root 使用当前树根，其他父标识通过页面查询定位。
// 查询失败直接返回错误，名称为空时返回空建议；上下文不参与该同步处理。
func (uc *SuggestSlugUseCase) Execute(_ context.Context, in SuggestSlugInput) (*SuggestSlugOutput, error) {
	if in.ParentID == "" || in.ParentID == "root" {
		return &SuggestSlugOutput{Slug: uc.slug.GenerateUniqueChildSlug(uc.tree.GetTree(), in.CurrentID, in.Title)}, nil
	}
	parent, err := uc.tree.FindPageByID(in.ParentID)
	if err != nil {
		return nil, err
	}
	return &SuggestSlugOutput{Slug: uc.slug.GenerateUniqueChildSlug(parent, in.CurrentID, in.Title)}, nil
}
