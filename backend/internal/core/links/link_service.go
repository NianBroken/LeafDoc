// 本文件协调页面树与链接索引。正文解析生成普通路径和双链标题目标，
// 存储层记录解析状态，查询结果再从当前页面树取得显示路径和目标标题。
package links

import (
	"context"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// LinkService 组合页面树及链接存储，本对象没有独立的后台任务或全流程互斥锁。
type LinkService struct {
	storageDir  string
	treeService *tree.TreeService
	store       *LinksStore
}

// NewLinkService 保存已有依赖，不在构造时读取正文或重建索引。
func NewLinkService(storageDir string, treeService *tree.TreeService, store *LinksStore) *LinkService {
	return &LinkService{
		storageDir:  storageDir,
		treeService: treeService,
		store:       store,
	}
}

// IndexAllPages 使用不可由请求取消的后台上下文重建索引。
func (b *LinkService) IndexAllPages() error {
	return b.IndexAllPagesContext(context.Background())
}

// IndexAllPagesContext 在树已加载时先清空索引，再收集节点并逐页重建出链。
// 中途取消或失败直接返回，前面完成的清空和页面写入不由此整体回滚。
func (b *LinkService) IndexAllPagesContext(ctx context.Context) error {
	if !b.treeService.IsLoaded() {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// 取消检查先于清空，后续数据库操作不直接接收此上下文。
	if err := b.store.Clear(); err != nil {
		return err
	}

	// 遍历节点只收集 ID，每个回调检查取消；正文随后通过批量页面入口读取。
	var ids []string
	if err := b.treeService.WalkNodes(func(id string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ids = append(ids, id)
		return nil
	}); err != nil {
		return err
	}

	// 返回的页面与错误按下标对应，任一读取或链接写入失败都会终止剩余处理。
	pages, errs := b.treeService.GetPages(ids)
	for i, page := range pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if errs[i] != nil {
			return errs[i]
		}
		targets := collectTargetsFromContent(b.treeService, page.CalculatePath(), page.Content)
		if err := b.store.AddLinks(page.ID, page.Title, targets); err != nil {
			return err
		}
	}

	return nil
}

// ClearLinks 删除全部链接记录，不改页面正文或页面树。
func (b *LinkService) ClearLinks() error {
	return b.store.Clear()
}

// GetBacklinksForPage 读取已解析入链，再加入该页面对应的歧义标题引用。
// 页面读取失败只让标题为空，不直接阻止索引查询；来源显示信息由结果转换取得。
func (b *LinkService) GetBacklinksForPage(pageID string) (*BacklinkResult, error) {
	pageTitle := ""
	if page, err := b.treeService.GetPage(pageID); err == nil {
		pageTitle = page.Title
	}

	backlinks, err := b.store.GetBacklinksForPage(pageID)
	if err != nil {
		return nil, err
	}
	backlinks, err = b.mergeAmbiguousWikiLinksIntoBacklinks(pageID, pageTitle, backlinks)
	if err != nil {
		return nil, err
	}
	return toBacklinkResult(b.treeService, backlinks), err
}

// GetOutgoingLinksForPage 读取并转换所有出链，存储错误与转换结果一同返回。
func (b *LinkService) GetOutgoingLinksForPage(pageID string) (*OutgoingResult, error) {
	outgoingLinks, err := b.store.GetOutgoingLinksForPage(pageID)
	return toOutgoingLinkResult(b.treeService, outgoingLinks), err
}

// GetRefactorMatchesForPrefix 直接查询旧路径及其下级路径的索引候选，不解析正文。
func (b *LinkService) GetRefactorMatchesForPrefix(oldPrefix string) ([]RefactorLinkMatch, error) {
	return b.store.GetRefactorMatchesForPrefix(oldPrefix)
}

// GetRefactorSourcePageIDsForPrefix 返回引用旧路径范围的来源 ID，由存储层去重。
func (b *LinkService) GetRefactorSourcePageIDsForPrefix(oldPrefix string) ([]string, error) {
	return b.store.GetRefactorSourcePageIDsForPrefix(oldPrefix)
}

// GetRefactorSourcePageIDsForWikiLinkTitle 按双链标题标记查找来源，不检查当前解析状态。
func (b *LinkService) GetRefactorSourcePageIDsForWikiLinkTitle(title string) ([]string, error) {
	return b.store.GetRefactorSourcePageIDsForWikiLinkTitle(title)
}

// UpdateRewrittenLinksAndHealForPages 从已有出链重建目标，而不是重新扫描正文。
// 它按规则转换路径和标题，重新解析目标，再替换这些页面的全部出链并处理入链恢复。
func (b *LinkService) UpdateRewrittenLinksAndHealForPages(pages []*tree.Page, rules []RewriteRule) error {
	outgoingByPageID, err := b.store.GetOutgoingLinksForPages(pageIDsForPages(pages))
	if err != nil {
		return err
	}

	// 空页面被跳过，来源标题和路径使用传入页面的当前值。
	updates := make([]PageLinkUpdate, 0, len(pages))
	for _, page := range pages {
		if page == nil {
			continue
		}
		pagePath := normalizeWikiPath(page.CalculatePath())
		targets := rewriteResolvedTargets(pagePath, outgoingByPageID[page.ID], rules, b.treeService)
		updates = append(updates, PageLinkUpdate{
			FromPageID: page.ID,
			FromTitle:  page.Title,
			ToPath:     pagePath,
			Targets:    targets,
		})
	}

	if len(updates) == 0 {
		return nil
	}

	return b.store.ReplaceLinksAndHeal(updates)
}

// GetLinkStatusForPage 分别查询正常入链、指向传入路径的失效入链及全部出链。
// 传入路径只做归一化，不在此验证它与 pageID 的当前路径一致。计数来自转换后条目数。
func (b *LinkService) GetLinkStatusForPage(pageID string, pagePath string) (*LinkStatusResult, error) {
	pagePath = normalizeWikiPath(pagePath)
	page, err := b.treeService.GetPage(pageID)
	if err != nil {
		return nil, err
	}

	// 正常入链来自已解析索引，并在标题多匹配时合并歧义双链来源。
	validBacklinks, err := b.store.GetBacklinksForPage(pageID)
	if err != nil {
		return nil, err
	}
	validBacklinks, err = b.mergeAmbiguousWikiLinksIntoBacklinks(pageID, page.Title, validBacklinks)
	if err != nil {
		return nil, err
	}
	validBacklinksResult := toBacklinkResult(b.treeService, validBacklinks)

	// 失效入链按目标路径精确查询，与上面的目标页面 ID 查询分开处理。
	brokenIncoming, err := b.store.GetBrokenIncomingForPath(pagePath)
	if err != nil {
		return nil, err
	}
	brokenIncomingResult := toBacklinkResult(b.treeService, brokenIncoming)

	// 全部出链按已保存状态分组，不在展示时重新解析正文。
	outgoings, err := b.store.GetOutgoingLinksForPage(pageID)
	if err != nil {
		return nil, err
	}
	// 只有 LinkBroken 进入失效组，其余状态进入正常出链组。
	okOut := make([]OutgoingResultItem, 0, len(outgoings))
	brokenOut := make([]OutgoingResultItem, 0)
	for _, outgoing := range outgoings {
		item := toOutgoingResultItem(b.treeService, outgoing)
		// 标题歧义单独保留，不等同于找不到目标；各匹配页的入链查询负责合并该来源。
		if outgoing.State == LinkBroken {
			brokenOut = append(brokenOut, item)
		} else {
			okOut = append(okOut, item)
		}
	}

	return &LinkStatusResult{
		Backlinks:       validBacklinksResult.Backlinks,
		BrokenIncoming:  brokenIncomingResult.Backlinks,
		Outgoings:       okOut,
		BrokenOutgoings: brokenOut,
		Counts: LinkStatusCounts{
			Backlinks:       len(validBacklinksResult.Backlinks),
			BrokenIncoming:  len(brokenIncomingResult.Backlinks),
			Outgoings:       len(okOut),
			BrokenOutgoings: len(brokenOut),
		},
	}, nil
}

// mergeAmbiguousWikiLinksIntoBacklinks 只在标题匹配多个页面且包含当前页面时查询
// 歧义引用。合并项的目标 ID 指向当前页面，失效标记设为 false。
// 原入链按原顺序保留，歧义项以来源 ID 和目标 ID 去重，不写回数据库。
func (b *LinkService) mergeAmbiguousWikiLinksIntoBacklinks(pageID string, pageTitle string, backlinks []Backlink) ([]Backlink, error) {
	if pageTitle == "" {
		return backlinks, nil
	}

	matches := b.treeService.FindPagesByTitle(pageTitle)
	if len(matches) <= 1 {
		return backlinks, nil
	}

	// 显式确认当前 ID 属于标题匹配集合，避免把同名引用归到无关页面。
	isMatchingPage := false
	for _, match := range matches {
		if match != nil && match.ID == pageID {
			isMatchingPage = true
			break
		}
	}
	if !isMatchingPage {
		return backlinks, nil
	}

	ambiguousRefs, err := b.store.GetAmbiguousIncomingForTitle(pageTitle)
	if err != nil {
		return nil, err
	}
	if len(ambiguousRefs) == 0 {
		return backlinks, nil
	}

	// 中间使用零字节隔开两个标识，区分由不同组成部分拼成的键。
	seen := make(map[string]struct{}, len(backlinks))
	merged := make([]Backlink, 0, len(backlinks)+len(ambiguousRefs))
	for _, backlink := range backlinks {
		key := backlink.FromPageID + "\x00" + backlink.ToPageID
		seen[key] = struct{}{}
		merged = append(merged, backlink)
	}

	for _, backlink := range ambiguousRefs {
		backlink.ToPageID = pageID
		backlink.Broken = false
		key := backlink.FromPageID + "\x00" + backlink.ToPageID
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, backlink)
	}

	return merged, nil
}

// UpdateLinksForPage 解析显式传入的正文，替换该页面全部出链。
// 页面对象只提供当前路径、ID 和标题，不要求 content 等于它的 Content 字段。
func (b *LinkService) UpdateLinksForPage(page *tree.Page, content string) error {
	targets := collectTargetsFromContent(b.treeService, page.CalculatePath(), content)
	return b.store.AddLinks(page.ID, page.Title, targets)
}

// UpdateLinksAndHealForPages 扫描每个非空页面的 Content，集中替换出链，
// 再把指向这些页面当前精确路径的失效记录恢复为已解析状态。
func (b *LinkService) UpdateLinksAndHealForPages(pages []*tree.Page) error {
	updates := make([]PageLinkUpdate, 0, len(pages))
	for _, page := range pages {
		if page == nil {
			continue
		}
		pagePath := normalizeWikiPath(page.CalculatePath())
		targets := collectTargetsFromContent(b.treeService, pagePath, page.Content)
		updates = append(updates, PageLinkUpdate{
			FromPageID: page.ID,
			FromTitle:  page.Title,
			ToPath:     pagePath,
			Targets:    targets,
		})
	}

	if len(updates) == 0 {
		return nil
	}

	return b.store.ReplaceLinksAndHeal(updates)
}

// DeleteOutgoingLinksForPage 删除页面作为来源的全部索引，不更改其他页面正文。
func (b *LinkService) DeleteOutgoingLinksForPage(pageID string) error {
	return b.store.DeleteOutgoingLinks(pageID)
}

// MarkIncomingLinksBrokenForPage 将指向目标 ID 的已解析记录置为失效并清空目标 ID。
func (b *LinkService) MarkIncomingLinksBrokenForPage(pageID string) error {
	return b.store.MarkIncomingLinksBroken(pageID)
}

// MarkLinksBrokenForPath 归一化目标路径，再把精确匹配的已解析记录置为失效。
func (b *LinkService) MarkLinksBrokenForPath(toPath string) error {
	toPath = normalizeWikiPath(toPath)
	return b.store.MarkLinksBrokenForPath(toPath)
}

// MarkLinksBrokenForPrefix 归一化路径后处理该路径及斜杠分隔的下级范围。
// 具体匹配由存储 SQL 的等值和 LIKE 条件执行，不在此重写正文。
func (b *LinkService) MarkLinksBrokenForPrefix(prefix string) error {
	prefix = normalizeWikiPath(prefix)
	return b.store.MarkLinksBrokenForPrefix(prefix)
}

// HealLinksForExactPath 把指向页面当前精确路径的失效记录绑定到该页面 ID。
func (b *LinkService) HealLinksForExactPath(page *tree.Page) error {
	toPath := normalizeWikiPath(page.CalculatePath())
	return b.store.HealLinksForPath(toPath, page.ID)
}

// ReconcileTitle 按当前树中的标题匹配数重算全部对应双链索引。
// 零个匹配为失效，一个匹配绑定目标 ID，多个匹配为歧义并清空目标 ID。
// 空白标题直接跳过，不读取来源正文，也不转换成普通路径查询。
func (b *LinkService) ReconcileTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}
	matches := b.treeService.FindPagesByTitle(title)
	pageID := ""
	if len(matches) == 1 {
		pageID = matches[0].ID
	}
	return b.store.ReconcileWikiLinksForTitle(title, classifyTitleMatches(len(matches)), pageID)
}

// Close 在存在存储时关闭它，空存储视为无需释放资源。
func (b *LinkService) Close() error {
	if b.store == nil {
		return nil
	}
	return b.store.Close()
}

// pageIDsForPages 依次收集非空页面 ID，保留输入顺序及重复 ID。
func pageIDsForPages(pages []*tree.Page) []string {
	ids := make([]string, 0, len(pages))
	for _, page := range pages {
		if page == nil {
			continue
		}
		ids = append(ids, page.ID)
	}
	return ids
}

// rewriteResolvedTargets 分开处理已有普通路径和双链标题标记，再按当前树重新解析。
// 普通路径应用路径规则，标题标记只应用标题规则，不把 wikilink: 当作页面路由。
// 返回值将替换页面全部出链；原状态不直接沿用，解析器会过滤无法形成目标的路径。
func rewriteResolvedTargets(currentPath string, outgoings []Outgoing, rules []RewriteRule, treeService *tree.TreeService) []TargetLink {
	if len(outgoings) == 0 {
		return nil
	}

	paths := make([]string, 0, len(outgoings))
	var wikiTitles []string
	for _, outgoing := range outgoings {
		if IsWikilinkSentinel(outgoing.ToPath) {
			wikiTitles = append(wikiTitles, rewriteWikiTitle(WikilinkTitleFromSentinel(outgoing.ToPath), rules))
			continue
		}
		targetPath := normalizeWikiPath(outgoing.ToPath)
		if rewritten, ok := applyRewriteRules(targetPath, rules); ok {
			targetPath = rewritten
		}
		paths = append(paths, targetPath)
	}

	result := resolveTargetLinks(treeService, currentPath, paths)
	return append(result, resolveWikiLinkTargets(treeService, wikiTitles)...)
}

// rewriteWikiTitle 按顺序选择首个旧标题非空、前后标题不同且忽略大小写匹配的规则。
// 命中后立即返回新标题，不把该结果继续交给后续规则；无命中时保留原值。
func rewriteWikiTitle(title string, rules []RewriteRule) string {
	for _, rule := range rules {
		if rule.OldTitle != "" && rule.OldTitle != rule.NewTitle && strings.EqualFold(rule.OldTitle, title) {
			return rule.NewTitle
		}
	}
	return title
}

// GetBrokenLinks 查询全部失效记录，再逐项读取来源页面的当前路径。
// 任一来源读取失败就返回错误，不返回已填充的部分列表。
func (b *LinkService) GetBrokenLinks() ([]BrokenLink, error) {
	links, err := b.store.GetBrokenLinks()
	if err != nil {
		return nil, err
	}

	for i := range links {
		page, err := b.treeService.GetPage(links[i].FromPageID)
		if err != nil {
			return nil, err
		}

		links[i].FromPath = page.CalculatePath()
	}

	return links, nil
}

// WithExecutor 返回绑定指定执行器和页面树视图的新服务，不替换原对象。
// 执行器的提交与回滚由拥有它的调用方负责。
func (s *LinkService) WithExecutor(executor database.Executor, boundTree *tree.TreeService) *LinkService {
	return &LinkService{storageDir: s.storageDir, treeService: boundTree, store: s.store.WithExecutor(executor)}
}
