// 本文件负责 Markdown 链接提取、双链标题解析、路由归一化及接口结果转换。
// 普通链接使用 Markdown 语法树，双链先用正则定位再排除不可处理区间。
package links

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// wikiLinkRe 匹配双中括号目标和可选别名，第一捕获组为标题或路径提示。
// 目标首字符必须非空白，因此开头带空格的 shell 条件表达式不按双链处理。
// 目标不包含井号或换行，别名不含结束方括号或换行。
var wikiLinkRe = regexp.MustCompile(`\[\[(\S[^\]|#\n]*?)(?:\|[^\]\n]+?)?\]\]`)

// wikilinkSentinelPrefix 在存储目标路径中标识按标题解析的双链，区别于普通路由。
// 标题唯一、缺失或歧义时都可使用此前缀，不只用于失效链接。
const wikilinkSentinelPrefix = "wikilink:"

// wikilinkSentinel 原样拼接目标文字，不执行裁空白或编码。
func wikilinkSentinel(target string) string {
	return wikilinkSentinelPrefix + target
}

// IsWikilinkSentinel 按区分大小写的前缀检查标题标记，不校验前缀后文字。
func IsWikilinkSentinel(toPath string) bool {
	return strings.HasPrefix(toPath, wikilinkSentinelPrefix)
}

// WikilinkTitleFromSentinel 去掉一次标题标记前缀，没有此前缀时原样返回输入。
func WikilinkTitleFromSentinel(toPath string) string {
	return strings.TrimPrefix(toPath, wikilinkSentinelPrefix)
}

// LinkState 表示链接索引中保存的目标解析状态，供查询和结果转换使用。
type LinkState int

const (
	// LinkResolved 表示解析得到唯一目标。
	LinkResolved LinkState = 0
	// LinkBroken 表示解析未找到目标。
	LinkBroken LinkState = 1
	// LinkAmbiguous 表示标题匹配不止一个页面，不能选出唯一目标，也不等同于失效。
	LinkAmbiguous LinkState = 2
)

// classifyTitleMatches 把匹配数零与一分别映射为失效和已解析，其余值映射为歧义。
// 正常调用传入切片长度，不会传入负数。
func classifyTitleMatches(matches int) LinkState {
	switch matches {
	case 0:
		return LinkBroken
	case 1:
		return LinkResolved
	default:
		return LinkAmbiguous
	}
}

// TargetLink 把解析到的页面 ID、存储目标路径和解析状态交给索引写入流程。
type TargetLink struct {
	TargetPageID   string
	TargetPagePath string
	State          LinkState
}

// markdownParser 为普通 Markdown 链接提取复用默认 Goldmark 解析配置。
var markdownParser = goldmark.New()

// isAssetLinkDestination 裁空白并移除单个尖括号包裹后，识别两种 assets 路径前缀。
// 判断区分大小写，不把任意外部地址中的 assets 片段当作站内资源。
func isAssetLinkDestination(dest string) bool {
	dest = strings.TrimSpace(dest)
	dest = strings.TrimPrefix(dest, "<")
	dest = strings.TrimSuffix(dest, ">")

	return strings.HasPrefix(dest, "/assets/") || strings.HasPrefix(dest, "assets/")
}

// extractLinksFromMarkdown 遍历普通链接节点，不把图片节点作为页面出链。
// 跳过 HTTP、HTTPS、mailto 和纯锚点，去除剩余目标的片段与查询串，再排除资源路径。
// 其他协议并未统一排除，重复链接保留，遍历失败时返回空列表。
func extractLinksFromMarkdown(content string) []string {
	links := []string{}
	reader := text.NewReader([]byte(content))
	doc := markdownParser.Parser().Parse(reader)

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if link, ok := n.(*ast.Link); ok && entering {
			// 只过滤这里明确列出的前缀，比较使用目标的小写副本。
			dest := string(link.Destination)
			lower := strings.ToLower(dest)
			if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:") || strings.HasPrefix(lower, "#") {
				return ast.WalkContinue, nil
			}
			// 先移除锚点，链接正文中目标的原始大小写保持原样。
			if idx := strings.Index(dest, "#"); idx != -1 {
				dest = dest[:idx]
			}
			// 再移除查询参数，索引只关心页面路径。
			if idx := strings.Index(dest, "?"); idx != -1 {
				dest = dest[:idx]
			}
			if isAssetLinkDestination(dest) {
				return ast.WalkContinue, nil
			}

			links = append(links, dest)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return []string{}
	}

	return links
}

// extractWikiLinksFromMarkdown 从双链匹配中提取裁去边缘空白的目标，别名不作为目标。
// 排除区间按匹配起始偏移判断，完全相同的目标只保留首次出现项。
// 没有正则匹配时返回 nil，有匹配但全部被过滤时返回空切片。
func extractWikiLinksFromMarkdown(content string) []string {
	matches := wikiLinkRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}
	excluded := NewMarkdownRefactorEngine().collectExcludedRanges(content)
	seen := make(map[string]struct{}, len(matches))
	targets := make([]string, 0, len(matches))
	for _, m := range matches {
		if isExcludedOffset(m[0], excluded) || m[2] == -1 {
			continue
		}
		target := strings.TrimSpace(content[m[2]:m[3]])
		if target == "" {
			continue
		}
		if _, dup := seen[target]; dup {
			continue
		}
		seen[target] = struct{}{}
		targets = append(targets, target)
	}
	return targets
}

// resolveWikiLinkTargets 要求页面树已加载，对每个目标先做标题查找。
// 唯一标题优先，即使标题包含斜杠也不转作路由；多标题匹配保留歧义。
// 仅在标题无匹配且目标含斜杠时回退路由查找，路由失败保存为普通失效路径。
func resolveWikiLinkTargets(treeService *tree.TreeService, targets []string) []TargetLink {
	if !treeService.IsLoaded() || len(targets) == 0 {
		return nil
	}

	var result []TargetLink
	for _, target := range targets {
		pages := treeService.FindPagesByTitle(target)

		// 目标保留原大小写，仅去掉一个开头斜杠，实际路由查找规则由树服务执行。
		if strings.Contains(target, "/") && len(pages) == 0 {
			routePath := strings.TrimPrefix(target, "/")
			page, err := treeService.FindPageByRoutePath(routePath)
			if err == nil && page != nil {
				result = append(result, TargetLink{
					TargetPageID:   page.ID,
					TargetPagePath: normalizeWikiPath(page.CalculatePath()),
					State:          LinkResolved,
				})
				continue
			}
			// 未命中的路径用开头斜杠保存，使按精确路径处理的索引流程能够识别它。
			result = append(result, TargetLink{
				State:          LinkBroken,
				TargetPagePath: "/" + routePath,
			})
			continue
		}

		link := TargetLink{
			State:          classifyTitleMatches(len(pages)),
			TargetPagePath: wikilinkSentinel(target),
		}
		if len(pages) == 1 {
			link.TargetPageID = pages[0].ID
		}
		result = append(result, link)
	}
	return result
}

// collectTargetsFromContent 分别解析普通链接与双链，再按此顺序合并结果。
// 两种语法之间不在此处统一去重。
func collectTargetsFromContent(treeService *tree.TreeService, pagePath string, content string) []TargetLink {
	mdLinks := extractLinksFromMarkdown(content)
	mdTargets := resolveTargetLinks(treeService, pagePath, mdLinks)

	wikiTargets := extractWikiLinksFromMarkdown(content)
	wikiResolved := resolveWikiLinkTargets(treeService, wikiTargets)

	return append(mdTargets, wikiResolved...)
}

// normalizeWikiPath 对非空输入移除片段与查询串、确保开头斜杠、折叠连续斜杠。
// 根路径保留单个斜杠，其他路径移除尾部斜杠；不裁空白或解析点路径段。
func normalizeWikiPath(p string) string {
	if p == "" {
		return ""
	}

	// 片段和查询串均不参与页面路径索引。
	if i := strings.Index(p, "#"); i != -1 {
		p = p[:i]
	}
	if i := strings.Index(p, "?"); i != -1 {
		p = p[:i]
	}

	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	// 循环替换直到不存在连续双斜杠，不把路径当作含协议的完整 URL。
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}

	// 只对非根路径移除尾部斜杠。
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}

	return p
}

// resolveURLPath 把当前页面路径作为目录基准解析相对链接，只返回目标 URL 的路径。
// example.com 仅用于 URL 解析基准，不产生网络请求，解析错误直接返回。
func resolveURLPath(currentPath, href string) (string, error) {
	currentPath = normalizeWikiPath(currentPath)

	// 基准结尾补斜杠，使 child 指向当前页面路径下的子路径。
	folderBase := currentPath
	if !strings.HasSuffix(folderBase, "/") {
		folderBase += "/"
	}

	base, err := url.Parse("https://example.com" + folderBase)
	if err != nil {
		return "", err
	}

	ref, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	resolved := base.ResolveReference(ref)

	// 只使用解码后的路径，不把目标主机、协议和查询参数写入结果。
	return normalizeWikiPath(resolved.Path), nil
}

// resolveTargetLinks 先排除资源，再解析普通链接路径并查询页面树。
// 树未加载返回 nil，路径解析失败、空路径和根路径均跳过，不产生失效记录。
// 查找失败统一保存为失效链接，成功保存页面 ID 与归一化路径。
func resolveTargetLinks(tree *tree.TreeService, currentPath string, links []string) []TargetLink {
	if !tree.IsLoaded() {
		return nil
	}

	var targetLinks []TargetLink

	for _, link := range links {
		if isAssetLinkDestination(link) {
			continue
		}

		// 相对目标按当前页作为目录解析，错误项不阻止其他链接处理。
		resolvedPath, err := resolveURLPath(currentPath, link)
		if err != nil || resolvedPath == "" {
			continue
		}

		// 树查找使用不带开头斜杠的路径，空值表示根路径并被跳过。
		normalizedForLookup := strings.TrimPrefix(resolvedPath, "/")
		if normalizedForLookup == "" {
			continue
		}

		// 当前树中存在对应页面才记为已解析。
		page, err := tree.FindPageByRoutePath(normalizedForLookup)
		if err == nil && page != nil {
			// 保存页面稳定 ID，避免后续结果只依赖显示路径。
			targetLinks = append(targetLinks, TargetLink{
				TargetPageID:   page.ID,
				TargetPagePath: resolvedPath,
				State:          LinkResolved,
			})
		} else {
			// 保留未解析路径供失效链接展示，目标 ID 保持为空。
			targetLinks = append(targetLinks, TargetLink{
				TargetPageID:   "",
				TargetPagePath: resolvedPath,
				State:          LinkBroken,
			})
		}
	}

	return targetLinks
}

// toBacklinkResult 逐项转换并统计条目数，不过滤转换后为空结构的条目。
func toBacklinkResult(tree *tree.TreeService, backlinks []Backlink) *BacklinkResult {
	var items []BacklinkResultItem
	for _, backlink := range backlinks {
		item := toBacklinkResultItem(tree, backlink)
		items = append(items, item)
	}
	return &BacklinkResult{
		Backlinks: items,
		Count:     len(items),
	}
}

// toBacklinkResultItem 从当前树读取来源路径，树未加载或来源查找失败返回零值结构。
// 来源标题沿用索引记录，不在此处用当前页面标题覆盖。
func toBacklinkResultItem(tree *tree.TreeService, backlink Backlink) BacklinkResultItem {
	if !tree.IsLoaded() {
		return BacklinkResultItem{}
	}

	page, err := tree.FindPageByID(backlink.FromPageID)
	if err != nil {
		return BacklinkResultItem{}
	}

	return BacklinkResultItem{
		FromPageID: backlink.FromPageID,
		FromTitle:  backlink.FromTitle,
		FromPath:   page.CalculatePath(),
		ToPageID:   backlink.ToPageID,
		Broken:     backlink.Broken,
	}
}

// toOutgoingLinkResult 按输入次序转换出链，Count 为实际追加条目数。
func toOutgoingLinkResult(tree *tree.TreeService, outgoings []Outgoing) *OutgoingResult {
	var items []OutgoingResultItem
	for _, outgoing := range outgoings {
		item := toOutgoingResultItem(tree, outgoing)
		items = append(items, item)
	}
	return &OutgoingResult{
		Outgoings: items,
		Count:     len(items),
	}
}

// toOutgoingResultItem 把标题标记还原为显示目标，并由已保存状态设置失效标志。
// 有目标 ID 且当前树能找到页面时填充当前标题，查找失败不重算链接状态。
func toOutgoingResultItem(tree *tree.TreeService, outgoing Outgoing) OutgoingResultItem {
	displayPath := outgoing.ToPath
	if IsWikilinkSentinel(outgoing.ToPath) {
		displayPath = WikilinkTitleFromSentinel(outgoing.ToPath)
	}
	item := OutgoingResultItem{
		ToPageID:   outgoing.ToPageID,
		ToPath:     displayPath,
		Broken:     outgoing.State == LinkBroken,
		FromPageID: outgoing.FromPageID,
	}

	if outgoing.ToPageID == "" {
		return item
	}

	if !tree.IsLoaded() {
		return item
	}

	toPage, err := tree.FindPageByID(outgoing.ToPageID)
	if err != nil || toPage == nil {
		return item
	}

	item.ToPageTitle = toPage.Title
	return item
}
