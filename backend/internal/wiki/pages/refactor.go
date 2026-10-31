package pages

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/links"
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// 重构类型决定目标路径计算及后续实际保存入口，不接受其他操作文字。
const (
	RefactorKindRename = "rename"
	RefactorKindMove   = "move"
)

// RefactorPreviewInput 描述重命名或移动方案，Content 为可选的目标页面正文。
// 预览只计算路径和链接影响，Content 不参与预览中的正文写入。
type RefactorPreviewInput struct {
	PageID      string
	Kind        string
	Title       string
	Slug        string
	Content     *string
	NewParentID *string
}

// RefactorPreview 返回旧新路径、外部受影响页面与汇总警告，不持久化这些计算结果。
type RefactorPreview struct {
	Kind          string                 `json:"kind"`
	PageID        string                 `json:"pageId"`
	OldPath       string                 `json:"oldPath"`
	NewPath       string                 `json:"newPath"`
	AffectedPages []RefactorAffectedPage `json:"affectedPages"`
	Counts        RefactorPreviewCounts  `json:"counts"`
	Warnings      []string               `json:"warnings"`
}

// RefactorPreviewCounts 保存受影响页面数和匹配计数，计数按索引匹配分支累计。
// MatchedLinks 不等于最终正文替换次数，也不等于 MatchedPaths 去重后的总长度。
type RefactorPreviewCounts struct {
	AffectedPages int `json:"affectedPages"`
	MatchedLinks  int `json:"matchedLinks"`
}

// RefactorAffectedPage 按来源页聚合匹配路径和双中括号语法，警告与匹配文字分别去重排序。
type RefactorAffectedPage struct {
	FromPageID   string   `json:"fromPageId"`
	FromTitle    string   `json:"fromTitle"`
	FromPath     string   `json:"fromPath"`
	MatchedPaths []string `json:"matchedPaths"`
	Warnings     []string `json:"warnings"`
}

// RefactorApplyInput 在预览输入上提供操作用户、预期版本、外部链接重写选择和位置。
type RefactorApplyInput struct {
	UserID  string
	Version string
	RefactorPreviewInput
	RewriteLinks bool
	// Position 仅传给移动入口，空值由移动用例转成 -1，最终排序沿用树服务规则。
	Position *int
}

// PreviewPageRefactorUseCase 读取页面及链接索引，计算目标路由与子树外的链接影响。
// 计算目标路径不代替实际移动时的版本、同级冲突和循环检查。
type PreviewPageRefactorUseCase struct {
	tree  *tree.TreeService
	slug  *tree.SlugService
	links *links.LinkService
	log   *slog.Logger
}

// NewPreviewPageRefactorUseCase 保存树、名称、链接和日志依赖，不预先计算方案。
func NewPreviewPageRefactorUseCase(
	t *tree.TreeService,
	s *tree.SlugService,
	l *links.LinkService,
	log *slog.Logger,
) *PreviewPageRefactorUseCase {
	return &PreviewPageRefactorUseCase{tree: t, slug: s, links: l, log: log}
}

// Execute 读取目标页面并计算路由，排除目标子树内来源页后生成影响清单。
// 未配置链接服务时返回空影响，查询或正文检查失败向上传递，上下文未参与同步查询。
func (uc *PreviewPageRefactorUseCase) Execute(_ context.Context, in RefactorPreviewInput) (*RefactorPreview, error) {
	page, err := uc.tree.GetPage(in.PageID)
	if err != nil {
		return nil, err
	}

	oldPath := page.CalculatePath()
	newPath, err := uc.computeTargetPath(page, in)
	if err != nil {
		return nil, err
	}

	excludeIDs := subtreeIDSet(page.PageNode)
	// 改标题的重命名还检查按旧标题记录的双中括号引用，不只检查路径前缀。
	affectedPages, matchedLinks, err := uc.getAffectedPages(oldPath, page.Title, excludeIDs, sentinelTitleForRefactor(page, in))
	if err != nil {
		return nil, err
	}

	return &RefactorPreview{
		Kind:          in.Kind,
		PageID:        in.PageID,
		OldPath:       oldPath,
		NewPath:       newPath,
		AffectedPages: affectedPages,
		Counts: RefactorPreviewCounts{
			AffectedPages: len(affectedPages),
			MatchedLinks:  matchedLinks,
		},
		Warnings: collectPreviewWarnings(affectedPages),
	}, nil
}

// computeTargetPath 对重命名验证标题和路径段，移动则读取目标父节点路由。
// 此方法只拼接路径，不检查新父节点类型、循环或同级名称是否被占用。
func (uc *PreviewPageRefactorUseCase) computeTargetPath(page *tree.Page, in RefactorPreviewInput) (string, error) {
	switch in.Kind {
	case RefactorKindRename:
		ve := sharederrors.NewValidationErrors()
		if in.Title == "" {
			ve.Add("title", "Title must not be empty")
		}
		if err := uc.slug.IsValidSlug(in.Slug); err != nil {
			ve.Add("slug", err.Error())
		}
		if ve.HasErrors() {
			return "", ve
		}
		parentPath := ""
		if page.Parent != nil {
			parentPath = page.Parent.CalculatePath()
		}
		if parentPath == "" {
			return "/" + in.Slug, nil
		}
		return parentPath + "/" + in.Slug, nil

	case RefactorKindMove:
		parentID := ""
		if in.NewParentID != nil {
			parentID = *in.NewParentID
		}
		parentPath, err := uc.resolveParentPath(parentID)
		if err != nil {
			return "", err
		}
		if parentPath == "" {
			return "/" + page.Slug, nil
		}
		return parentPath + "/" + page.Slug, nil

	default:
		return "", fmt.Errorf("unsupported refactor kind: %s", in.Kind)
	}
}

// resolveParentPath 将空标识和 root 解释为根路由，其余标识读取实际页面路径。
func (uc *PreviewPageRefactorUseCase) resolveParentPath(parentID string) (string, error) {
	if parentID == "" || parentID == "root" {
		return "", nil
	}
	parent, err := uc.tree.GetPage(parentID)
	if err != nil {
		return "", err
	}
	return parent.CalculatePath(), nil
}

// getAffectedPages 先按旧路径前缀查询索引，再按来源页分组并排除给定子树标识。
// 标题引用按独立规则纳入，随后读取每页正文取得实际双中括号文字和重写警告。
// 最终列表按来源标题、路径排序，任一正文读取失败会使整个预览失败。
func (uc *PreviewPageRefactorUseCase) getAffectedPages(oldPath string, pageTitle string, excludeIDs map[string]struct{}, sentinelTitle string) ([]RefactorAffectedPage, int, error) {
	if uc.links == nil {
		return nil, 0, nil
	}
	matches, err := uc.links.GetRefactorMatchesForPrefix(oldPath)
	if err != nil {
		return nil, 0, err
	}

	grouped := make(map[string]*RefactorAffectedPage)
	totalMatches := 0
	for _, match := range matches {
		if _, excluded := excludeIDs[match.FromPageID]; excluded {
			continue
		}
		fromPath := ""
		if page, err := uc.tree.GetPage(match.FromPageID); err == nil && page != nil {
			fromPath = page.CalculatePath()
		}
		item, ok := grouped[match.FromPageID]
		if !ok {
			item = &RefactorAffectedPage{
				FromPageID: match.FromPageID,
				FromTitle:  match.FromTitle,
				FromPath:   fromPath,
			}
			grouped[match.FromPageID] = item
		}
		if !containsString(item.MatchedPaths, match.ToPath) {
			item.MatchedPaths = append(item.MatchedPaths, match.ToPath)
		}
		totalMatches++
	}

	// 按标题记录的引用不由路径前缀查询直接覆盖，改标题时单独查找这些来源页。
	if sentinelTitle != "" {
		// 多个页面共用旧标题时引用存在歧义，不将其纳入此重构的标题重写列表。
		// 这里只检查当前同名数量，不预测重命名后最终会解析到哪个页面。
		if len(uc.tree.FindPagesByTitle(sentinelTitle)) <= 1 {
			sentinelIDs, err := uc.links.GetRefactorSourcePageIDsForWikiLinkTitle(sentinelTitle)
			if err != nil {
				return nil, 0, err
			}
			wikiLinkSyntax := "[[" + sentinelTitle + "]]"
			for _, id := range sentinelIDs {
				if _, excluded := excludeIDs[id]; excluded {
					continue
				}
				if _, already := grouped[id]; already {
					// 已由路径匹配登记的来源页不重复增加计数，正文语法在后续统一检查。
					continue
				}
				fromPath := ""
				fromTitle := ""
				if p, err := uc.tree.GetPage(id); err == nil && p != nil {
					fromPath = p.CalculatePath()
					fromTitle = p.Title
				}
				grouped[id] = &RefactorAffectedPage{
					FromPageID:   id,
					FromTitle:    fromTitle,
					FromPath:     fromPath,
					MatchedPaths: []string{wikiLinkSyntax},
				}
				totalMatches++
			}
		}
	}

	engine := links.NewMarkdownRefactorEngine()
	finder := links.CompileWikiLinkFinder(oldPath, pageTitle)
	items := make([]RefactorAffectedPage, 0, len(grouped))
	for _, item := range grouped {
		sourcePage, err := uc.tree.GetPage(item.FromPageID)
		if err != nil {
			return nil, 0, err
		}

		// 从正文取得匹配的双中括号表达式，加入显示清单并去掉精确的旧路径条目。
		// 其他不同目标路径条目保留，不因同页存在双中括号引用而全部移除。
		wikiLinks := engine.FindWikiLinksPrecompiled(sourcePage.Content, finder)
		for _, wl := range wikiLinks {
			if !containsString(item.MatchedPaths, wl) {
				item.MatchedPaths = append(item.MatchedPaths, wl)
			}
			// 仅移除与 oldPath 完全相同的项，语法显示不改变前面累计的匹配数。
			item.MatchedPaths = removeString(item.MatchedPaths, oldPath)
		}

		// 同路径规则用于收集重写引擎的警告，生成的正文不在预览中保存。
		rules := []links.RewriteRule{{OldPath: oldPath, NewPath: oldPath}}
		result := engine.Rewrite(sourcePage.Content, sourcePage.CalculatePath(), rules)
		for _, w := range result.Warnings {
			if !containsString(item.Warnings, w.Message) {
				item.Warnings = append(item.Warnings, w.Message)
			}
		}
		sort.Strings(item.MatchedPaths)
		sort.Strings(item.Warnings)
		item.MatchedPaths = ensureStrings(item.MatchedPaths)
		item.Warnings = ensureStrings(item.Warnings)
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].FromTitle == items[j].FromTitle {
			return items[i].FromPath < items[j].FromPath
		}
		return items[i].FromTitle < items[j].FromTitle
	})
	return items, totalMatches, nil
}

// ApplyPageRefactorUseCase 在树工作单元内重命名或移动，并处理页面内部与外部链接。
// RewriteLinks 控制子树外来链重写，路径变化子树的相对链接处理仍会执行。
type ApplyPageRefactorUseCase struct {
	tree     *tree.TreeService
	slug     *tree.SlugService
	revision *revision.Service
	links    *links.LinkService
	log      *slog.Logger
	preview  *PreviewPageRefactorUseCase
	metrics  *httpmetrics.HTTPMetrics
}

// NewApplyPageRefactorUseCase 保存依赖并建立共享同一组查询服务的预览用例。
func NewApplyPageRefactorUseCase(
	t *tree.TreeService,
	s *tree.SlugService,
	r *revision.Service,
	l *links.LinkService,
	log *slog.Logger,
	metrics *httpmetrics.HTTPMetrics,
) *ApplyPageRefactorUseCase {
	return &ApplyPageRefactorUseCase{
		tree:     t,
		slug:     s,
		revision: r,
		links:    l,
		log:      log,
		preview:  NewPreviewPageRefactorUseCase(t, s, l, log),
		metrics:  metrics,
	}
}

// Execute 先在绑定工作单元中建立应用计划和正文快照，再实际修改目标页面。
// 目标修改通过版本校验后才重写其他页面，任一返回错误传播到外层事务。
// 此处建立的保存编排器只包含链接与历史处理，不包含搜索、标签或属性处理器。
func (uc *ApplyPageRefactorUseCase) Execute(ctx context.Context, in RefactorApplyInput) (*tree.Page, error) {
	if !uc.tree.InUnitOfWork() {
		var result *tree.Page
		err := uc.tree.WithUnitOfWork(ctx, func(bound *tree.TreeService, exec database.Executor) error {
			copy := *uc
			copy.tree = bound
			copy.revision = uc.revision.WithExecutor(exec, bound)
			if uc.links != nil {
				copy.links = uc.links.WithExecutor(exec, bound)
			}
			copy.preview = NewPreviewPageRefactorUseCase(bound, uc.slug, copy.links, uc.log)
			var inner error
			result, inner = copy.Execute(ctx, in)
			return inner
		})
		return result, err
	}

	started := time.Now()
	plan, err := uc.buildApplyPlan(in)
	if err != nil {
		return nil, err
	}

	snapshots, err := uc.captureSnapshots(plan.page, in)
	if err != nil {
		return nil, err
	}

	// 在节点字段变化前保存规则中的旧路径和旧标题，实际应用在目标写入成功之后。
	incomingRule := links.RewriteRule{OldPath: plan.oldPath, NewPath: plan.newPath}
	if in.Kind == RefactorKindRename && in.Title != plan.page.Title {
		incomingRule.OldTitle = plan.page.Title
		incomingRule.NewTitle = in.Title
	}

	o := pagesave.NewPageSaveOrchestrator(
		uc.metrics,
		pagesave.NewLinkIndexSideEffect(uc.links, uc.log, uc.metrics),
		pagesave.NewRevisionSideEffect(uc.revision, uc.log, uc.metrics),
	)

	switch in.Kind {
	case RefactorKindRename:
		updateStepStarted := time.Now()
		updateUC := NewUpdatePageUseCase(uc.tree, uc.slug, o, uc.log, uc.metrics)
		updated, err := updateUC.Execute(ctx, UpdatePageInput{
			UserID:  in.UserID,
			ID:      in.PageID,
			Version: in.Version,
			Title:   in.Title,
			Slug:    in.Slug,
			Content: in.Content,
			Kind:    kindPage(),
		})
		uc.metrics.ObserveRefactorStep(in.Kind, "update_target_page", updateStepStarted)
		if err != nil {
			return nil, err
		}
		return uc.finishPathChange(in, plan, snapshots, incomingRule, updated.Page.ID, started)

	case RefactorKindMove:
		parentID := ""
		if in.NewParentID != nil {
			parentID = *in.NewParentID
		}
		moveStepStarted := time.Now()
		moveUC := NewMovePageUseCase(uc.tree, o, uc.log, uc.metrics)
		err := moveUC.Execute(ctx, MovePageInput{UserID: in.UserID, ID: in.PageID, Version: in.Version, ParentID: parentID, Position: in.Position})
		uc.metrics.ObserveRefactorStep(in.Kind, "move_target_page", moveStepStarted)
		if err != nil {
			return nil, err
		}
		return uc.finishPathChange(in, plan, snapshots, incomingRule, in.PageID, started)

	default:
		return nil, fmt.Errorf("unsupported refactor kind: %s", in.Kind)
	}
}

// finishPathChange 在目标保存之后处理共有步骤，按选择重写外部来链，再处理子树链接。
// 外部重写启用时还刷新来源页链接状态，末尾重新读取目标页并记录成功指标。
// 任一步失败立即返回，不继续后续阶段。
func (uc *ApplyPageRefactorUseCase) finishPathChange(in RefactorApplyInput, plan *applyRefactorPlan, snapshots []pathChangeSnapshot, incomingRule links.RewriteRule, pageID string, started time.Time) (*tree.Page, error) {
	if in.RewriteLinks {
		stepStarted := time.Now()
		err := uc.rewriteAffectedPages(in.UserID, plan.affectedPageIDs, []links.RewriteRule{incomingRule})
		uc.metrics.ObserveRefactorStep(in.Kind, "rewrite_affected_pages", stepStarted)
		if err != nil {
			return nil, err
		}
	}
	subtreeStepStarted := time.Now()
	err := uc.rewritePathChangedSubtree(in.UserID, snapshots, plan.oldPath, plan.newPath)
	uc.metrics.ObserveRefactorStep(in.Kind, "rewrite_path_changed_subtree", subtreeStepStarted)
	if err != nil {
		return nil, err
	}
	if in.RewriteLinks {
		refreshStepStarted := time.Now()
		err := uc.refreshAffectedPageLinks(plan.affectedPageIDs)
		uc.metrics.ObserveRefactorStep(in.Kind, "refresh_affected_links", refreshStepStarted)
		if err != nil {
			return nil, err
		}
	}
	page, err := uc.tree.GetPage(pageID)
	if err == nil {
		uc.metrics.ObserveRefactor(in.Kind, in.RewriteLinks, plan.affectedPages, plan.matchedLinks, started)
	}
	return page, err
}

// applyRefactorPlan 保存变动前的页面与旧新路由，以及外部来源页标识和统计值。
type applyRefactorPlan struct {
	page            *tree.Page
	oldPath         string
	newPath         string
	affectedPageIDs []string
	affectedPages   int
	matchedLinks    int
}

// buildApplyPlan 读取目标并复用预览的路径计算，只有启用外部重写且有链接服务才查来源。
// 来源列表排除目标子树，按标题引用的来源在满足唯一标题条件时单独合并。
func (uc *ApplyPageRefactorUseCase) buildApplyPlan(in RefactorApplyInput) (*applyRefactorPlan, error) {
	page, err := uc.tree.GetPage(in.PageID)
	if err != nil {
		return nil, err
	}

	oldPath := page.CalculatePath()
	newPath, err := uc.preview.computeTargetPath(page, in.RefactorPreviewInput)
	if err != nil {
		return nil, err
	}

	plan := &applyRefactorPlan{
		page:    page,
		oldPath: oldPath,
		newPath: newPath,
	}

	if !in.RewriteLinks || uc.links == nil {
		return plan, nil
	}

	affectedPages, matchedLinks, err := uc.preview.getAffectedPages(oldPath, page.Title, subtreeIDSet(page.PageNode), sentinelTitleForRefactor(page, in.RefactorPreviewInput))
	if err != nil {
		return nil, err
	}
	plan.affectedPages = len(affectedPages)
	plan.matchedLinks = matchedLinks

	pageIDs, err := uc.links.GetRefactorSourcePageIDsForPrefix(oldPath)
	if err != nil {
		return nil, err
	}

	excludeIDs := subtreeIDSet(page.PageNode)
	plan.affectedPageIDs = make([]string, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		if _, excluded := excludeIDs[pageID]; excluded {
			continue
		}
		plan.affectedPageIDs = append(plan.affectedPageIDs, pageID)
	}

	// 改标题时还查询按旧标题标记的来源，路径前缀查询不直接覆盖这些索引记录。
	if in.Kind == RefactorKindRename && in.Title != plan.page.Title {
		// 与预览采用相同同名数量限制，有歧义的标题引用不进入外部正文重写列表。
		if len(uc.tree.FindPagesByTitle(plan.page.Title)) <= 1 {
			sentinelIDs, err := uc.links.GetRefactorSourcePageIDsForWikiLinkTitle(plan.page.Title)
			if err != nil {
				return nil, err
			}
			for _, id := range sentinelIDs {
				if _, excluded := excludeIDs[id]; excluded {
					continue
				}
				if !containsString(plan.affectedPageIDs, id) {
					plan.affectedPageIDs = append(plan.affectedPageIDs, id)
				}
			}
		}
	}

	return plan, nil
}

// sentinelTitleForRefactor 只为实际改变标题的重命名返回旧标题，其余操作返回空串。
func sentinelTitleForRefactor(page *tree.Page, in RefactorPreviewInput) string {
	if in.Kind == RefactorKindRename && in.Title != page.Title {
		return page.Title
	}
	return ""
}

// pathChangeSnapshot 在树结构变化前保存页面路径和正文，供相对链接按旧基准解析。
// RootPage 标识方案目标页，快照记录该字段但本文件重写流程不据此选择分支。
type pathChangeSnapshot struct {
	PageID   string
	OldPath  string
	Content  string
	RootPage bool
}

// captureSnapshots 按子树顺序读取全部正文，任一读取失败都不进入目标修改。
// 目标页明确提供 Content 时快照使用该输入，其他页面沿用当前正文。
func (uc *ApplyPageRefactorUseCase) captureSnapshots(page *tree.Page, in RefactorApplyInput) ([]pathChangeSnapshot, error) {
	ids := collectSubtreeIDs(page.PageNode)
	if len(ids) == 0 {
		ids = []string{in.PageID}
	}
	pages, errs := uc.tree.GetPages(ids)
	snapshots := make([]pathChangeSnapshot, 0, len(ids))
	for i, p := range pages {
		if errs[i] != nil {
			return nil, errs[i]
		}
		content := p.Content
		if ids[i] == in.PageID && in.Content != nil {
			content = *in.Content
		}
		snapshots = append(snapshots, pathChangeSnapshot{
			PageID: p.ID, OldPath: p.CalculatePath(), Content: content, RootPage: ids[i] == in.PageID,
		})
	}
	return snapshots, nil
}

// rewriteAffectedPages 先处理 Markdown 链接，再以预编译规则处理双中括号链接。
// 读取失败来源被跳过，没有替换计数或正文没有变化时不写入。
// 保存成功后批量刷新重写页面链接状态，读取告警之外的返回错误中止流程。
func (uc *ApplyPageRefactorUseCase) rewriteAffectedPages(userID string, affectedPageIDs []string, rules []links.RewriteRule) error {
	engine := links.NewMarkdownRefactorEngine()
	compiledWikiRewrites := links.CompileWikiLinkRewrites(rules)

	uc.log.Debug("rewriting wiki-links in affected pages", "pages", len(affectedPageIDs), "rules", len(rules))

	var rewrites []contentRewrite
	pagesByID := uc.loadPagesByID(affectedPageIDs, "failed to get page for link rewrite, skipping")

	for _, pageID := range affectedPageIDs {
		page, ok := pagesByID[pageID]
		if !ok {
			continue
		}
		mdResult := engine.Rewrite(page.Content, page.CalculatePath(), rules)
		wikiResult := engine.RewriteWikiLinksPrecompiled(mdResult.Content, compiledWikiRewrites)
		newContent := wikiResult.Content
		if mdResult.Count() == 0 && wikiResult.Count() == 0 || newContent == page.Content {
			continue
		}
		rewrites = append(rewrites, contentRewrite{page: page, content: newContent})
	}

	if len(rewrites) == 0 {
		return nil
	}

	updatedPages, err := uc.persistContentRewrites(userID, rewrites)
	if err != nil {
		return err
	}

	if uc.links != nil && len(updatedPages) > 0 {
		if err := uc.links.UpdateRewrittenLinksAndHealForPages(updatedPages, rules); err != nil {
			return err
		}
	}
	return nil
}

// contentRewrite 关联待写页面与计算后的纯正文，页面指针用于取得身份。
type contentRewrite struct {
	page    *tree.Page
	content string
}

// persistContentRewrites 批量提交正文，再按输入顺序检查错误并重新读取页面。
// 所有写入调用结束后才检查首个错误，不能把返回错误理解为后续项尚未尝试。
// 页面均读取成功后批量记录正文历史，任一失败返回给外层工作单元。
func (uc *ApplyPageRefactorUseCase) persistContentRewrites(userID string, rewrites []contentRewrite) ([]*tree.Page, error) {
	bulk := make([]tree.BulkContentUpdate, 0, len(rewrites))
	for _, r := range rewrites {
		bulk = append(bulk, tree.BulkContentUpdate{ID: r.page.ID, Content: r.content})
	}

	errs := uc.tree.BulkUpdateContent(userID, bulk)
	updatedPages := make([]*tree.Page, 0, len(rewrites))
	for i, r := range rewrites {
		if errs[i] != nil {
			return nil, errs[i]
		}
		page, err := uc.tree.GetPage(r.page.ID)
		if err != nil {
			return nil, err
		}
		updatedPages = append(updatedPages, page)
	}

	if uc.revision != nil {
		revErrs := uc.revision.RecordContentUpdates(updatedPages, userID, "")
		for i, err := range revErrs {
			if err != nil {
				return nil, fmt.Errorf("record revision for %s: %w", updatedPages[i].ID, err)
			}
		}
	}
	return updatedPages, nil
}

// refreshAffectedPageLinks 重新读取成功的来源页面并批量更新链接，失效来源只记录告警。
// 没有链接服务、输入为空或全部读取失败时直接成功。
func (uc *ApplyPageRefactorUseCase) refreshAffectedPageLinks(pageIDs []string) error {
	if uc.links == nil || len(pageIDs) == 0 {
		return nil
	}

	pagesByID := uc.loadPagesByID(pageIDs, "failed to get page for link refresh, skipping")
	currentPages := make([]*tree.Page, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		page, ok := pagesByID[pageID]
		if !ok {
			continue
		}
		currentPages = append(currentPages, page)
	}

	if len(currentPages) == 0 {
		return nil
	}

	return uc.links.UpdateLinksAndHealForPages(currentPages)
}

// rewritePathChangedSubtree 对照旧正文与旧路径，在当前路径下重建子树内链接地址。
// 每页读取失败则跳过，实际正文有变化才进入批量保存和历史记录。
// 此流程不由 RewriteLinks 开关关闭，开关只控制子树外来源页的重写。
func (uc *ApplyPageRefactorUseCase) rewritePathChangedSubtree(userID string, snapshots []pathChangeSnapshot, oldPath, newPath string) error {
	engine := links.NewMarkdownRefactorEngine()
	rules := []links.RewriteRule{{OldPath: oldPath, NewPath: newPath}}

	var rewrites []contentRewrite

	pageIDs := make([]string, 0, len(snapshots))
	for _, snap := range snapshots {
		pageIDs = append(pageIDs, snap.PageID)
	}
	pagesByID := uc.loadPagesByID(pageIDs, "failed to get subtree page for link rewrite, skipping")

	for _, snap := range snapshots {
		current, ok := pagesByID[snap.PageID]
		if !ok {
			continue
		}
		currentPath := current.CalculatePath()
		// 用快照中的旧路径解释相对链接，再以当前路径计算应写入的相对地址。
		relResult := engine.RewriteRelativeLinksForPathChange(snap.Content, snap.OldPath, currentPath, rules)
		// 第一轮结果仍包含旧路径文字时，再用旧新路径规则执行普通 Markdown 重写。
		// 文字包含判断只决定是否调用第二轮，不代替重写引擎的实际链接识别。
		finalContent := relResult.Content
		if strings.Contains(relResult.Content, oldPath) {
			finalContent = engine.Rewrite(relResult.Content, currentPath, rules).Content
		}
		if finalContent == current.Content {
			continue
		}
		rewrites = append(rewrites, contentRewrite{page: current, content: finalContent})
	}

	if len(rewrites) == 0 {
		return nil
	}

	updatedPages, err := uc.persistContentRewrites(userID, rewrites)
	if err != nil {
		return err
	}

	if uc.links != nil && len(updatedPages) > 0 {
		if err := uc.links.UpdateLinksAndHealForPages(updatedPages); err != nil {
			return err
		}
	}
	return nil
}

// subtreeIDSet 把子树标识列表转成集合，供外部来源筛选排除目标子树。
func subtreeIDSet(node *tree.PageNode) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, id := range collectSubtreeIDs(node) {
		ids[id] = struct{}{}
	}
	return ids
}

// containsString 按字符串精确相等检查成员，不折叠大小写或清理路径。
func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

// removeString 复用原切片底层存储，保留不等于目标的全部项及相对顺序。
func removeString(values []string, target string) []string {
	out := values[:0]
	for _, v := range values {
		if v != target {
			out = append(out, v)
		}
	}
	return out
}

// ensureStrings 将 nil 列表转成非 nil 空列表，使响应编码为空数组而非 null。
func ensureStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// loadPagesByID 批量读取页面并按标识建表，错误项记录指定告警，空页面跳过。
// 重复标识以最后读取结果覆盖，不为失败项返回占位页面。
func (uc *ApplyPageRefactorUseCase) loadPagesByID(ids []string, warningMessage string) map[string]*tree.Page {
	if len(ids) == 0 {
		return map[string]*tree.Page{}
	}

	pages, errs := uc.tree.GetPages(ids)
	loaded := make(map[string]*tree.Page, len(ids))
	for i, id := range ids {
		if errs[i] != nil {
			uc.log.Warn(warningMessage, "pageID", id, "error", errs[i])
			continue
		}
		if pages[i] == nil {
			continue
		}
		loaded[id] = pages[i]
	}

	return loaded
}

// collectPreviewWarnings 对各页警告文字精确去重并排序，没有警告也返回非 nil 列表。
func collectPreviewWarnings(pages []RefactorAffectedPage) []string {
	var warnings []string
	for _, p := range pages {
		for _, w := range p.Warnings {
			if !containsString(warnings, w) {
				warnings = append(warnings, w)
			}
		}
	}
	sort.Strings(warnings)
	return ensureStrings(warnings)
}

// kindPage 返回指向普通页面类型值的指针，供更新输入的可选类型字段使用。
func kindPage() *tree.NodeKind {
	k := tree.NodeKindPage
	return &k
}
