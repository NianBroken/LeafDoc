// 本文件解析 Markdown 中的链接，按路径或标题规则计算文本替换区间。
// 普通链接先经语法树确认目标，再扫描原文取得字节位置；双链使用预编译正则，
// 两者均借助语法树排除代码内容，不访问页面树或验证目标页面是否实际存在。
package links

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownRefactorEngine 复用默认 Goldmark 解析器，保留原正文格式而不重新渲染整篇文档。
type MarkdownRefactorEngine struct {
	parser goldmark.Markdown
}

// RewriteWarning 描述未能映射到支持语法或无法生成目标的条目，不作为函数错误返回。
type RewriteWarning struct {
	Message string
}

// RewriteReplacement 用原正文的字节偏移表示左闭右开的替换区间。
// NewValue 只替换该区间，不要求与旧文本字节长度相同。
type RewriteReplacement struct {
	Start    int
	End      int
	NewValue string
}

// RewriteResult 同时返回处理后的正文、实际替换计划及去重后的警告。
// 没有替换时保留原文，警告仍可单独存在。
type RewriteResult struct {
	Content      string
	Replacements []RewriteReplacement
	Warnings     []RewriteWarning
}

// rewriteCandidate 来自语法树，只保存解析后的链接目标，不保存原文位置。
type rewriteCandidate struct {
	Destination string
}

// inlineLinkOccurrence 来自原文扫描，目标区间不包含尖括号、标签或链接标题。
type inlineLinkOccurrence struct {
	RawDestination string
	Start          int
	End            int
}

// NewMarkdownRefactorEngine 创建使用默认语法设置的解析器，不读取外部配置。
func NewMarkdownRefactorEngine() *MarkdownRefactorEngine {
	return &MarkdownRefactorEngine{
		parser: goldmark.New(),
	}
}

// Count 返回替换计划条目数，不统计单个目标文本中的字符变化数量。
func (r RewriteResult) Count() int {
	return len(r.Replacements)
}

// wikiRewrite 将一个已编译匹配式与其目标文本配对，别名从匹配原文中保留。
type wikiRewrite struct {
	re        *regexp.Regexp
	newTarget string
}

// CompiledWikiLinkRewrites 保存规则对应的预编译匹配式，可供多篇正文重复使用。
// 同一规则的标题匹配排在路径提示匹配之前，影响重叠区间保留哪一项。
type CompiledWikiLinkRewrites struct {
	rewrites []wikiRewrite
}

// CompileWikiLinkRewrites 按规则顺序生成标题和路径提示匹配式，规则文字经过正则转义。
// 标题忽略大小写，路径提示区分大小写；路径只移除归一化后的开头斜杠，
// 匹配目标前后允许空白，不把路径规则扩展成任意子路径的双链匹配。
func CompileWikiLinkRewrites(rules []RewriteRule) CompiledWikiLinkRewrites {
	var rw []wikiRewrite
	for _, rule := range rules {
		if rule.OldTitle != "" && rule.OldTitle != rule.NewTitle {
			rw = append(rw, wikiRewrite{
				// 此处允许目标两边空白，重写后不会保留这些边缘空白。
				// 提取器另有目标首字符限制，不能据此认定两者接受的语法完全相同。
				re:        regexp.MustCompile(`(?i)\[\[\s*` + regexp.QuoteMeta(rule.OldTitle) + `\s*(\|[^\]\n]*)?\]\]`),
				newTarget: rule.NewTitle,
			})
		}
		oldHint := strings.TrimPrefix(normalizeWikiPath(rule.OldPath), "/")
		newHint := strings.TrimPrefix(normalizeWikiPath(rule.NewPath), "/")
		if oldHint != "" && oldHint != newHint {
			rw = append(rw, wikiRewrite{
				re:        regexp.MustCompile(`\[\[\s*` + regexp.QuoteMeta(oldHint) + `\s*(\|[^\]\n]*)?\]\]`),
				newTarget: newHint,
			})
		}
	}
	return CompiledWikiLinkRewrites{rewrites: rw}
}

// RewriteWikiLinks 编译规则后重写双链标题与路径提示，保留已有别名内容。
// 每次调用都会编译规则，多页处理可直接使用预编译入口复用匹配式。
func (e *MarkdownRefactorEngine) RewriteWikiLinks(content string, rules []RewriteRule) RewriteResult {
	return e.RewriteWikiLinksPrecompiled(content, CompileWikiLinkRewrites(rules))
}

// RewriteWikiLinksPrecompiled 在原文上收集全部匹配，排除起点位于代码区间的项。
// 匹配基于同一份原文，替换结果不继续参与其他规则的匹配，不产生链式重写。
func (e *MarkdownRefactorEngine) RewriteWikiLinksPrecompiled(content string, compiled CompiledWikiLinkRewrites) RewriteResult {
	if content == "" {
		return RewriteResult{Content: content}
	}
	if len(compiled.rewrites) == 0 {
		return RewriteResult{Content: content}
	}

	excludedRanges := e.collectExcludedRanges(content)

	// 可选捕获组保存竖线及别名，重建时只替换目标文字。
	var replacements []RewriteReplacement
	for _, rw := range compiled.rewrites {
		for _, m := range rw.re.FindAllStringSubmatchIndex(content, -1) {
			if isExcludedOffset(m[0], excludedRanges) {
				continue
			}
			var newValue string
			if m[2] >= 0 {
				newValue = "[[" + rw.newTarget + content[m[2]:m[3]] + "]]"
			} else {
				newValue = "[[" + rw.newTarget + "]]"
			}
			replacements = append(replacements, RewriteReplacement{
				Start: m[0], End: m[1], NewValue: newValue,
			})
		}
	}

	if len(replacements) == 0 {
		return RewriteResult{Content: content}
	}

	sort.SliceStable(replacements, func(i, j int) bool {
		return replacements[i].Start < replacements[j].Start
	})

	// 按起点稳定排序后去除重叠匹配。同一起点保留规则收集顺序中较早的一项，
	// 因而同规则的标题匹配优先于覆盖相同区间的路径提示匹配。
	i := 0
	for _, r := range replacements {
		if i > 0 && r.Start < replacements[i-1].End {
			continue
		}
		replacements[i] = r
		i++
	}
	replacements = replacements[:i]

	return RewriteResult{
		Content:      applyReplacements(content, replacements),
		Replacements: replacements,
	}
}

// CompiledWikiLinkFinder 复用预览用的路径和标题匹配式，保存标准化展示文本。
type CompiledWikiLinkFinder struct{ patterns []wikiFindPattern }

// wikiFindPattern 的 syntax 是预览标签，不是每个匹配在原文中的完整写法。
type wikiFindPattern struct {
	re     *regexp.Regexp
	syntax string
}

// CompileWikiLinkFinder 先构造路径提示匹配，再构造忽略大小写的标题匹配。
// 两项不合并去重，即使标准展示文本相同也会各自保存。
func CompileWikiLinkFinder(oldPath, pageTitle string) CompiledWikiLinkFinder {
	finder := CompiledWikiLinkFinder{}
	oldHint := strings.TrimPrefix(normalizeWikiPath(oldPath), "/")
	if oldHint != "" {
		finder.patterns = append(finder.patterns, wikiFindPattern{
			re: regexp.MustCompile(`\[\[\s*` + regexp.QuoteMeta(oldHint) + `\s*(?:\|[^\]\n]*)?\]\]`), syntax: "[[" + oldHint + "]]",
		})
	}
	if pageTitle != "" {
		finder.patterns = append(finder.patterns, wikiFindPattern{
			re: regexp.MustCompile(`(?i)\[\[\s*` + regexp.QuoteMeta(pageTitle) + `\s*(?:\|[^\]\n]*)?\]\]`), syntax: "[[" + pageTitle + "]]",
		})
	}
	return finder
}

// FindWikiLinksForPath 编译给定路径和标题的预览匹配式，查找代码区间之外的引用。
// 返回值是每种匹配式的标准标签，不包含原文别名、空白或全部出现位置。
func (e *MarkdownRefactorEngine) FindWikiLinksForPath(content, oldPath, pageTitle string) []string {
	if content == "" {
		return nil
	}
	return e.FindWikiLinksPrecompiled(content, CompileWikiLinkFinder(oldPath, pageTitle))
}

// FindWikiLinksPrecompiled 每种匹配式只要找到一个未排除项就追加一次标准标签。
// 结果按匹配式顺序排列，不按原文位置排列，也不跨匹配式去重。
func (e *MarkdownRefactorEngine) FindWikiLinksPrecompiled(content string, finder CompiledWikiLinkFinder) []string {
	if content == "" || len(finder.patterns) == 0 {
		return nil
	}
	excludedRanges := e.collectExcludedRanges(content)
	var found []string
	for _, pattern := range finder.patterns {
		for _, m := range pattern.re.FindAllStringIndex(content, -1) {
			if !isExcludedOffset(m[0], excludedRanges) {
				found = append(found, pattern.syntax)
				break
			}
		}
	}
	return found
}

// RewriteRelativeLinksForPathChange 在来源页路径变化时重算普通相对链接。
// 先按旧页面路径解析原目标，再应用目标规则，按新页面路径生成相对地址。
// 来源路径文本完全相同就直接返回，即使规则包含目标变化也不执行此入口的处理。
func (e *MarkdownRefactorEngine) RewriteRelativeLinksForPathChange(content string, oldCurrentPath string, newCurrentPath string, rules []RewriteRule) RewriteResult {
	if content == "" || oldCurrentPath == newCurrentPath {
		return RewriteResult{Content: content}
	}

	candidates, excludedRanges := e.collectCandidatesAndExcludedRanges(content)
	if len(candidates) == 0 {
		return RewriteResult{Content: content}
	}

	occurrences := scanInlineLinkOccurrences(content, excludedRanges)
	replacements, warnings := buildPathChangeRewritePlan(oldCurrentPath, newCurrentPath, rules, candidates, occurrences)
	if len(replacements) == 0 {
		return RewriteResult{
			Content:  content,
			Warnings: warnings,
		}
	}

	return RewriteResult{
		Content:      applyReplacements(content, replacements),
		Replacements: replacements,
		Warnings:     warnings,
	}
}

// Rewrite 只重写普通链接中命中路径规则的目标部分，保留标签、标题及其他正文。
// 语法树确认链接身份，原文扫描定位可替换区间，无法对应的候选以警告返回。
func (e *MarkdownRefactorEngine) Rewrite(content string, currentPath string, rules []RewriteRule) RewriteResult {
	if len(rules) == 0 || content == "" {
		return RewriteResult{Content: content}
	}

	candidates, excludedRanges := e.collectCandidatesAndExcludedRanges(content)
	if len(candidates) == 0 {
		return RewriteResult{Content: content}
	}

	occurrences := scanInlineLinkOccurrences(content, excludedRanges)
	replacements, warnings := buildRewritePlan(currentPath, rules, candidates, occurrences)
	if len(replacements) == 0 {
		return RewriteResult{
			Content:  content,
			Warnings: warnings,
		}
	}

	return RewriteResult{
		Content:      applyReplacements(content, replacements),
		Replacements: replacements,
		Warnings:     warnings,
	}
}

// collectExcludedRanges 复用完整解析过程，只取其中的代码排除区间。
func (e *MarkdownRefactorEngine) collectExcludedRanges(content string) []textRange {
	_, excluded := e.collectCandidatesAndExcludedRanges(content)
	return excluded
}

// collectCandidatesAndExcludedRanges 遍历进入节点事件，收集普通链接目标和代码内容区间。
// 图片节点不作为候选，排除范围来自行内代码文本及围栏、缩进代码块的内容行，
// 不在此把所有 HTML 或文档其他结构统一列为排除区间。
func (e *MarkdownRefactorEngine) collectCandidatesAndExcludedRanges(content string) ([]rewriteCandidate, []textRange) {
	reader := text.NewReader([]byte(content))
	doc := e.parser.Parser().Parse(reader)

	var candidates []rewriteCandidate
	var excluded []textRange

	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := node.(type) {
		case *ast.Link:
			candidates = append(candidates, rewriteCandidate{
				Destination: string(n.Destination),
			})
		case *ast.CodeSpan:
			excluded = append(excluded, collectTextNodeRanges(n)...)
		case *ast.FencedCodeBlock:
			excluded = append(excluded, collectBlockRanges(n)...)
		case *ast.CodeBlock:
			excluded = append(excluded, collectBlockRanges(n)...)
		}

		return ast.WalkContinue, nil
	})

	return candidates, mergeRanges(excluded)
}

// collectTextNodeRanges 只读取直接子节点中的文本片段，不递归任意后代。
func collectTextNodeRanges(parent ast.Node) []textRange {
	var ranges []textRange
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			ranges = append(ranges, textRange{
				Start: textNode.Segment.Start,
				Stop:  textNode.Segment.Stop,
			})
		}
	}
	return ranges
}

// collectBlockRanges 将代码块各内容行的字节范围逐项保存，空行集合返回空结果。
func collectBlockRanges(node interface{ Lines() *text.Segments }) []textRange {
	lines := node.Lines()
	if lines == nil {
		return nil
	}

	ranges := make([]textRange, 0, lines.Len())
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		ranges = append(ranges, textRange{
			Start: segment.Start,
			Stop:  segment.Stop,
		})
	}
	return ranges
}

// textRange 使用左闭右开的字节区间，Stop 位置本身不属于该区间。
type textRange struct {
	Start int
	Stop  int
}

// mergeRanges 按起点及终点原地排序输入，合并重叠或首尾相接的区间。
// 返回有序且互不相接的范围，供偏移检查遇到后续区间时提前结束。
func mergeRanges(ranges []textRange) []textRange {
	if len(ranges) < 2 {
		return ranges
	}

	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Start == ranges[j].Start {
			return ranges[i].Stop < ranges[j].Stop
		}
		return ranges[i].Start < ranges[j].Start
	})

	merged := make([]textRange, 0, len(ranges))
	current := ranges[0]
	for i := 1; i < len(ranges); i++ {
		next := ranges[i]
		if next.Start <= current.Stop {
			if next.Stop > current.Stop {
				current.Stop = next.Stop
			}
			continue
		}
		merged = append(merged, current)
		current = next
	}
	merged = append(merged, current)
	return merged
}

// scanInlineLinkOccurrences 按字节寻找方括号标签及其后的圆括号目标，
// 跳过代码区间中的起点和紧邻感叹号的图片起点。这里只扫描行内目标语法，
// 不定位引用式链接定义；候选是否为真实 Markdown 链接由后续语法树映射确认。
func scanInlineLinkOccurrences(content string, excluded []textRange) []inlineLinkOccurrence {
	var occurrences []inlineLinkOccurrence
	for i := 0; i < len(content); i++ {
		if isExcludedOffset(i, excluded) {
			continue
		}
		if content[i] != '[' {
			continue
		}
		if i > 0 && content[i-1] == '!' {
			continue
		}

		labelEnd := findClosingBracket(content, i)
		if labelEnd == -1 {
			continue
		}

		j := labelEnd + 1
		for j < len(content) && (content[j] == ' ' || content[j] == '\n' || content[j] == '\t') {
			j++
		}
		if j >= len(content) || content[j] != '(' {
			continue
		}

		occurrence, ok := parseInlineLinkOccurrence(content, j+1)
		if !ok {
			continue
		}
		// 记录目标区间后继续从其后扫描，不重新扫描该目标内部的字符。
		occurrences = append(occurrences, occurrence)
		i = occurrence.End
	}
	return occurrences
}

// parseInlineLinkOccurrence 从链接左圆括号后的偏移开始读取目标，跳过空格、换行和制表。
// 尖括号目标读取到右尖括号，普通目标识别嵌套圆括号及反斜杠跳过，
// 再检查整段存在结束圆括号；返回区间只覆盖非空目标，不包含标题文字。
func parseInlineLinkOccurrence(content string, start int) (inlineLinkOccurrence, bool) {
	i := start
	for i < len(content) && (content[i] == ' ' || content[i] == '\n' || content[i] == '\t') {
		i++
	}
	if i >= len(content) {
		return inlineLinkOccurrence{}, false
	}

	destStart := i
	destEnd := -1

	if content[i] == '<' {
		i++
		destStart = i
		for i < len(content) {
			if content[i] == '>' {
				destEnd = i
				break
			}
			if content[i] == '\\' {
				i++
			}
			i++
		}
	} else {
		depth := 0
		for i < len(content) {
			switch content[i] {
			case '\\':
				i += 2
				continue
			case '(':
				depth++
			case ')':
				if depth == 0 {
					destEnd = i
					goto done
				}
				depth--
			case ' ', '\n', '\t':
				destEnd = i
				goto done
			}
			i++
		}
	}

done:
	if destEnd == -1 || destEnd <= destStart {
		return inlineLinkOccurrence{}, false
	}

	closingParen := findClosingParen(content, start)
	if closingParen == -1 {
		return inlineLinkOccurrence{}, false
	}

	return inlineLinkOccurrence{
		RawDestination: content[destStart:destEnd],
		Start:          destStart,
		End:            destEnd,
	}, true
}

// buildRewritePlan 按语法树候选顺序向前消耗原文扫描结果，比较规范化后的目标文本。
// 命中后计算路径替换，找不到对应位置则记录不支持语法警告；扫描游标不会回退，
// 所以没有映射的候选也会消耗其搜索经过的位置。
func buildRewritePlan(currentPath string, rules []RewriteRule, candidates []rewriteCandidate, occurrences []inlineLinkOccurrence) ([]RewriteReplacement, []RewriteWarning) {
	var replacements []RewriteReplacement
	var warnings []RewriteWarning

	occurrenceIndex := 0
	for _, candidate := range candidates {
		mapped := false
		for occurrenceIndex < len(occurrences) {
			occurrence := occurrences[occurrenceIndex]
			occurrenceIndex++

			if normalizeCandidateDestination(occurrence.RawDestination) != normalizeCandidateDestination(candidate.Destination) {
				continue
			}

			newDest, changed, warning := rewriteLinkDestination(currentPath, occurrence.RawDestination, rules)
			if warning != nil {
				warnings = append(warnings, *warning)
			}
			if changed {
				replacements = append(replacements, RewriteReplacement{
					Start:    occurrence.Start,
					End:      occurrence.End,
					NewValue: newDest,
				})
			}
			mapped = true
			break
		}

		if !mapped {
			warnings = append(warnings, RewriteWarning{
				Message: fmt.Sprintf("Skipped unsupported link syntax for destination %q", candidate.Destination),
			})
		}
	}

	return replacements, dedupeWarnings(warnings)
}

// buildPathChangeRewritePlan 使用同样的单向候选映射，但按来源页移动规则重算相对地址。
// 原文中不需要变化的匹配仍算成功映射，不为它生成替换或不支持语法警告。
func buildPathChangeRewritePlan(oldCurrentPath string, newCurrentPath string, rules []RewriteRule, candidates []rewriteCandidate, occurrences []inlineLinkOccurrence) ([]RewriteReplacement, []RewriteWarning) {
	var replacements []RewriteReplacement
	var warnings []RewriteWarning

	occurrenceIndex := 0
	for _, candidate := range candidates {
		mapped := false
		for occurrenceIndex < len(occurrences) {
			occurrence := occurrences[occurrenceIndex]
			occurrenceIndex++

			if normalizeCandidateDestination(occurrence.RawDestination) != normalizeCandidateDestination(candidate.Destination) {
				continue
			}

			newDest, changed, warning := rewriteRelativeLinkForPathChange(oldCurrentPath, newCurrentPath, occurrence.RawDestination, rules)
			if warning != nil {
				warnings = append(warnings, *warning)
			}
			if changed {
				replacements = append(replacements, RewriteReplacement{
					Start:    occurrence.Start,
					End:      occurrence.End,
					NewValue: newDest,
				})
			}
			mapped = true
			break
		}

		if !mapped {
			warnings = append(warnings, RewriteWarning{
				Message: fmt.Sprintf("Skipped unsupported link syntax for destination %q", candidate.Destination),
			})
		}
	}

	return replacements, dedupeWarnings(warnings)
}

// normalizeCandidateDestination 裁边缘空白并移除各一个尖括号，
// 不解码实体、百分号或反斜杠转义，供语法树目标与原文目标比较。
func normalizeCandidateDestination(destination string) string {
	if destination == "" {
		return ""
	}
	destination = strings.TrimSpace(destination)
	destination = strings.TrimPrefix(destination, "<")
	destination = strings.TrimSuffix(destination, ">")
	return destination
}

// rewriteLinkDestination 分离查询串和片段后解析普通目标，只处理命中路径规则的链接。
// 原目标以斜杠开头时输出绝对路径，否则按来源页经规则转换后的路径计算相对地址，
// 再原样拼回后缀。外部、附件和空目标保持原文，无法解析或生成空相对地址时给出警告。
func rewriteLinkDestination(currentPath string, destination string, rules []RewriteRule) (string, bool, *RewriteWarning) {
	baseDest, suffix := splitLinkDestination(destination)
	if baseDest == "" || isExternalLinkDestination(baseDest) || isAssetLinkDestination(baseDest) {
		return destination, false, nil
	}

	resolvedPath, err := resolveURLPath(currentPath, baseDest)
	if err != nil || resolvedPath == "" {
		return destination, false, &RewriteWarning{
			Message: fmt.Sprintf("Skipped unresolved link destination %q", destination),
		}
	}

	newResolvedPath, ok := applyRewriteRules(resolvedPath, rules)
	if !ok || newResolvedPath == resolvedPath {
		return destination, false, nil
	}

	var rewrittenBase string
	if strings.HasPrefix(baseDest, "/") {
		rewrittenBase = newResolvedPath
	} else {
		currentPathForRelative := currentPath
		if nextCurrentPath, rewrittenCurrentPath := applyRewriteRules(normalizeWikiPath(currentPath), rules); rewrittenCurrentPath {
			currentPathForRelative = nextCurrentPath
		}
		rewrittenBase = relativeWikiLinkPath(currentPathForRelative, newResolvedPath)
	}

	if rewrittenBase == "" {
		return destination, false, &RewriteWarning{
			Message: fmt.Sprintf("Skipped empty rewritten destination for %q", destination),
		}
	}

	return rewrittenBase + suffix, true, nil
}

// rewriteRelativeLinkForPathChange 只处理非绝对、非外部且非附件的相对目标。
// 目标即使未命中规则，也按来源的新位置重新计算相对路径，保留查询串和片段。
// 新旧目标文本相同时不产生替换，目标等于来源导致空相对路径时保留原文并给出警告。
func rewriteRelativeLinkForPathChange(oldCurrentPath string, newCurrentPath string, destination string, rules []RewriteRule) (string, bool, *RewriteWarning) {
	baseDest, suffix := splitLinkDestination(destination)
	if baseDest == "" || strings.HasPrefix(baseDest, "/") || isExternalLinkDestination(baseDest) || isAssetLinkDestination(baseDest) {
		return destination, false, nil
	}

	resolvedPath, err := resolveURLPath(oldCurrentPath, baseDest)
	if err != nil || resolvedPath == "" {
		return destination, false, &RewriteWarning{
			Message: fmt.Sprintf("Skipped unresolved link destination %q", destination),
		}
	}

	targetPath := resolvedPath
	if rewrittenTarget, ok := applyRewriteRules(resolvedPath, rules); ok {
		targetPath = rewrittenTarget
	}

	rewrittenBase := relativeWikiLinkPath(newCurrentPath, targetPath)
	if rewrittenBase == "" {
		return destination, false, &RewriteWarning{
			Message: fmt.Sprintf("Skipped empty rewritten destination for %q", destination),
		}
	}

	if rewrittenBase == baseDest {
		return destination, false, nil
	}

	return rewrittenBase + suffix, true, nil
}

// applyReplacements 按传入顺序拼接未改动区间与新文本，跳过起点落在已消耗范围内的项。
// 调用方负责提供有效且按位置排序的区间；此函数不排序，也不检查超出正文的偏移。
func applyReplacements(content string, replacements []RewriteReplacement) string {
	if len(replacements) == 0 {
		return content
	}

	var builder strings.Builder
	last := 0
	for _, replacement := range replacements {
		if replacement.Start < last {
			continue
		}
		builder.WriteString(content[last:replacement.Start])
		builder.WriteString(replacement.NewValue)
		last = replacement.End
	}
	builder.WriteString(content[last:])
	return builder.String()
}

// dedupeWarnings 按完整消息文本去重，保留第一次出现的顺序。
func dedupeWarnings(warnings []RewriteWarning) []RewriteWarning {
	if len(warnings) < 2 {
		return warnings
	}

	seen := make(map[string]struct{}, len(warnings))
	deduped := make([]RewriteWarning, 0, len(warnings))
	for _, warning := range warnings {
		if _, ok := seen[warning.Message]; ok {
			continue
		}
		seen[warning.Message] = struct{}{}
		deduped = append(deduped, warning)
	}
	return deduped
}

// isExcludedOffset 判断字节是否处于任一排除区间。输入范围按起点排序，
// 偏移早于当前范围时可以直接判定未命中，不继续遍历更后的范围。
func isExcludedOffset(offset int, ranges []textRange) bool {
	for _, current := range ranges {
		if offset >= current.Start && offset < current.Stop {
			return true
		}
		if offset < current.Start {
			return false
		}
	}
	return false
}

// splitLinkDestination 在最早的问号或井号处分离基础目标与后缀，后缀原样保留。
func splitLinkDestination(destination string) (string, string) {
	if destination == "" {
		return "", ""
	}
	if idx := strings.IndexAny(destination, "?#"); idx != -1 {
		return destination[:idx], destination[idx:]
	}
	return destination, ""
}

// isExternalLinkDestination 忽略大小写识别 HTTP、HTTPS、mailto 和纯锚点开头。
// 未列出的其他协议不会仅因含冒号而被本函数排除。
func isExternalLinkDestination(destination string) bool {
	lower := strings.ToLower(destination)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "#")
}

// applyRewriteRules 按输入顺序使用首个精确路径或斜杠下级前缀匹配。
// 下级路径保留旧前缀后的文本，不执行最长前缀选择、连续规则应用或规则归一化。
func applyRewriteRules(resolvedPath string, rules []RewriteRule) (string, bool) {
	for _, rule := range rules {
		if resolvedPath == rule.OldPath {
			return rule.NewPath, true
		}
		if strings.HasPrefix(resolvedPath, rule.OldPath+"/") {
			return rule.NewPath + strings.TrimPrefix(resolvedPath, rule.OldPath), true
		}
	}
	return "", false
}

// relativeWikiLinkPath 将来源页面路径视为目录，找出共同路径段后，
// 为来源剩余段各写一个 ..，再追加目标剩余段。比较区分大小写，不做 URL 编码。
// 空目标及目标等于来源都返回空文本，不用点号表示同目录。
func relativeWikiLinkPath(currentPath string, targetPath string) string {
	base := normalizeWikiPath(currentPath)
	target := normalizeWikiPath(targetPath)
	if target == "" {
		return ""
	}

	baseParts := splitWikiPathSegments(base)
	targetParts := splitWikiPathSegments(target)

	common := 0
	for common < len(baseParts) && common < len(targetParts) && baseParts[common] == targetParts[common] {
		common++
	}

	var relParts []string
	for i := common; i < len(baseParts); i++ {
		relParts = append(relParts, "..")
	}
	relParts = append(relParts, targetParts[common:]...)

	rel := strings.Join(relParts, "/")
	if rel == "" {
		return ""
	}
	return rel
}

// splitWikiPathSegments 归一化斜杠并去掉首尾斜杠，根路径返回 nil。
// 点路径段在这个辅助函数中不折叠。
func splitWikiPathSegments(value string) []string {
	normalized := strings.Trim(normalizeWikiPath(value), "/")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "/")
}

// findClosingBracket 从左方括号起按嵌套深度寻找对应右方括号，
// 反斜杠使下一个字节不参与括号计数，直到末尾仍未闭合则返回负一。
func findClosingBracket(content string, start int) int {
	depth := 0
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		case '\\':
			i++
		}
	}
	return -1
}

// findClosingParen 从左圆括号之后开始，以初始深度一寻找对应右圆括号。
// 内部圆括号改变深度，反斜杠跳过下一字节，不单独解析标题引号。
func findClosingParen(content string, start int) int {
	depth := 1
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		case '\\':
			i++
		}
	}
	return -1
}
