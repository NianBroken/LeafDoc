package importer

import (
	"net/url"

	"path"
	"path/filepath"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// indexFilename 是目录形式链接查找的默认 Markdown 文件名。
const indexFilename = "index.md"

// importTarget 保存计划中源文件对应的页面路径和节点类型，不表示页面已经创建成功。
type importTarget struct {
	targetPath string
	kind       tree.NodeKind
}

// contentTransformer 在一次导入中共享页面查找表和附件上传缓存。
// 页面路径来自计划，附件文件来自解压目录；正文转换会调用 Wiki 接口上传附件。
type contentTransformer struct {
	sourceBasePath string
	assetMaxBytes  int64
	slugger        *tree.SlugService
	// 源路径采用小写键，名称表与路径后缀表保存候选列表，以便拒绝有歧义的匹配。
	pagesBySource   map[string]importTarget
	pagesByBasename map[string][]string
	pagesBySuffix   map[string][]string
	assetUploads    map[string]string
}

// newContentTransformer 遍历全部计划项，建立源路径、文件名称和目标路径后缀的索引。
// 这些索引记录计划中的目标，不读取页面树，也不检查各项后续是否执行成功。
func newContentTransformer(plan *PlanResult, sourceBasePath string, assetMaxBytes int64) *contentTransformer {
	pagesBySource := make(map[string]importTarget, len(plan.Items))
	pagesByBasename := make(map[string][]string, len(plan.Items))
	pagesBySuffix := make(map[string][]string, len(plan.Items))
	for _, item := range plan.Items {
		normalizedSource := normalizePlanSourcePath(item.SourcePath)
		pagesBySource[normalizedSource] = importTarget{
			targetPath: item.TargetPath,
			kind:       item.Kind,
		}

		// index 文件不进入按名称匹配的表，避免多个目录首页形成同名候选。
		if basenameKey := normalizePageBasenameForLookup(path.Base(normalizedSource)); basenameKey != "" {
			pagesByBasename[basenameKey] = append(pagesByBasename[basenameKey], item.TargetPath)
		}

		for _, suffixKey := range buildTargetPathSuffixKeys(item.TargetPath) {
			pagesBySuffix[suffixKey] = append(pagesBySuffix[suffixKey], item.TargetPath)
		}
	}

	return &contentTransformer{
		sourceBasePath:  sourceBasePath,
		assetMaxBytes:   assetMaxBytes,
		slugger:         tree.NewSlugService(),
		pagesBySource:   pagesBySource,
		pagesByBasename: pagesByBasename,
		pagesBySuffix:   pagesBySuffix,
		assetUploads:    map[string]string{},
	}
}

// TransformContent 对一页正文依次处理 Markdown 链接和双中括号链接。
// 两轮处理分别按本模块的扫描规则保留行内代码和围栏代码区域。
// 页面链接映射到计划路径，能读取的本地附件通过 Wiki 接口上传；错误中止转换。
func (t *contentTransformer) TransformContent(
	userID string,
	sourcePath string,
	page *tree.Page,
	content string,
	wiki ImporterWiki,
) (string, error) {
	rewritten, err := rewriteOutsideCodeSpans(content, func(segment string) (string, error) {
		return t.rewriteMarkdownLinks(userID, sourcePath, page, segment, wiki)
	})
	if err != nil {
		return "", err
	}
	return rewriteOutsideCodeSpans(rewritten, func(segment string) (string, error) {
		return t.rewriteWikiLinks(userID, sourcePath, page, segment, wiki)
	})
}

// rewriteMarkdownLinks 扫描普通链接与图片标记，用第一个右方括号定位标签末尾。
// 只有标签紧接左圆括号时才解析目标，目标内部用圆括号深度寻找结束位置。
// 缺少闭合标记时保留剩余文字，不处理反斜杠转义或完整 Markdown 语法规则。
func (t *contentTransformer) rewriteMarkdownLinks(
	userID string,
	sourcePath string,
	page *tree.Page,
	content string,
	wiki ImporterWiki,
) (string, error) {
	var out strings.Builder
	for i := 0; i < len(content); i++ {
		if content[i] != '[' && (content[i] != '!' || i+1 >= len(content) || content[i+1] != '[') {
			out.WriteByte(content[i])
			continue
		}

		start := i
		isImage := false
		if content[i] == '!' {
			isImage = true
			i++
		}

		labelEnd := strings.IndexByte(content[i:], ']')
		if labelEnd < 0 {
			out.WriteString(content[start:])
			return out.String(), nil
		}
		labelEnd += i
		if labelEnd+1 >= len(content) || content[labelEnd+1] != '(' {
			out.WriteString(content[start : labelEnd+1])
			i = labelEnd
			continue
		}

		destStart := labelEnd + 2
		destEnd := findMarkdownLinkDestinationEnd(content, destStart)
		if destEnd < 0 {
			out.WriteString(content[start:])
			return out.String(), nil
		}

		// 标签和图片前缀原样写回，仅将圆括号内的目标交给路径解析流程。
		destination := content[destStart:destEnd]
		rewritten, err := t.rewriteDestination(userID, sourcePath, page, destination, wiki)
		if err != nil {
			return "", err
		}

		if isImage {
			out.WriteByte('!')
		}
		out.WriteString(content[i : labelEnd+2])
		out.WriteString(rewritten)
		out.WriteByte(')')
		i = destEnd
	}

	return out.String(), nil
}

// rewriteDestination 拆出目标地址、尖括号包装和地址后的文字，再解析本地目标。
// 未解析成功时返回整个原字符串；解析成功时保留包装和后缀，但移除目标外侧空白。
func (t *contentTransformer) rewriteDestination(
	userID string,
	sourcePath string,
	page *tree.Page,
	destination string,
	wiki ImporterWiki,
) (string, error) {
	trimmed := strings.TrimSpace(destination)
	if trimmed == "" {
		return destination, nil
	}

	prefix, href, suffix := splitMarkdownDestination(trimmed)
	href = normalizeImportedHref(href)
	resolved, _, err := t.resolveDestination(userID, sourcePath, page, href, wiki)
	if err != nil {
		return "", err
	}
	if resolved == "" {
		return destination, nil
	}
	return prefix + resolved + suffix, nil
}

// resolveDestination 先查找导入计划中的页面，未命中时尝试读取并上传本地附件。
// 查询串和片段从目标中分离，在成功路径之后原样接回。布尔返回值只表示附件命中。
// 空地址、识别出的外部地址和纯片段不进入本地查找，返回空结果供调用方处理。
func (t *contentTransformer) resolveDestination(
	userID string,
	sourcePath string,
	page *tree.Page,
	href string,
	wiki ImporterWiki,
) (string, bool, error) {
	rawTarget, suffix := splitURLSuffix(href)
	if rawTarget == "" || isExternalHref(rawTarget) || strings.HasPrefix(rawTarget, "#") {
		return "", false, nil
	}
	rawTarget = decodeImportTarget(rawTarget)

	if targetPath, ok := t.resolvePagePath(sourcePath, rawTarget); ok {
		return "/" + targetPath + suffix, false, nil
	}

	assetPath, err := t.resolveAndUploadAsset(userID, sourcePath, page, rawTarget, wiki)
	if err != nil {
		return "", false, err
	}
	if assetPath != "" {
		return assetPath + suffix, true, nil
	}

	return "", false, nil
}

// resolvePagePath 只在导入计划的查找表中匹配页面，不查询现有 Wiki 页面树。
// 查找顺序为源文件候选路径、唯一文件名称、唯一目标路径后缀。
func (t *contentTransformer) resolvePagePath(sourcePath string, href string) (string, bool) {
	candidates := buildSourceCandidates(sourcePath, href)
	if !strings.HasPrefix(href, "/") && !strings.HasPrefix(href, ".") {
		candidates = append(candidates, buildSourceCandidates(sourcePath, "/"+href)...)
	}
	for _, candidate := range candidates {
		if target, ok := t.pagesBySource[normalizePlanSourcePath(candidate)]; ok {
			return target.targetPath, true
		}
	}

	// 不带路径的目标允许按文件名称查找，去重后必须恰好对应一个目标页面。
	// 带点前缀或斜杠的链接不走这条名称回退路径。
	if basenameKey, ok := basenameOnlyLookupKey(href); ok {
		if matches := uniqueStrings(t.pagesByBasename[basenameKey]); len(matches) == 1 {
			return matches[0], true
		}
	}

	if match, ok := t.resolveUniqueTargetPathSuffix(href); ok {
		return match, true
	}

	return "", false
}

// resolveUniqueTargetPathSuffix 将目标各路径段转换为页面路径段后匹配后缀索引。
// 非 Markdown 附件被排除，路径必须含斜杠，去重后必须只有一个目标。
func (t *contentTransformer) resolveUniqueTargetPathSuffix(href string) (string, bool) {
	if isNonMarkdownAssetTarget(href) {
		return "", false
	}

	suffix, ok := t.normalizeWikiHrefToRoutePath(href)
	if !ok || suffix == "" || !strings.Contains(suffix, "/") {
		return "", false
	}

	matches := uniqueStrings(t.pagesBySuffix[suffix])
	if len(matches) != 1 {
		return "", false
	}
	return matches[0], true
}

// normalizePlanSourcePath 清理路径、移除一个前导斜杠并转换为小写查找键。
// 该键只用于计划内匹配，不读取磁盘，也不改变原文件路径。
func normalizePlanSourcePath(p string) string {
	return strings.ToLower(path.Clean(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(p)), "/")))
}

// basenameOnlyLookupKey 只接受无斜杠且不以点开头的目标，再提取名称查找键。
func basenameOnlyLookupKey(href string) (string, bool) {
	trimmed := strings.TrimSpace(decodeImportTarget(href))
	if trimmed == "" || strings.Contains(trimmed, "/") || strings.HasPrefix(trimmed, ".") {
		return "", false
	}

	key := normalizePageBasenameForLookup(trimmed)
	return key, key != ""
}

// normalizePageBasenameForLookup 去除查询串、片段、目录和 Markdown 扩展名。
// 空名称与 index 不参与名称索引，其余名称去除两端空白后转成小写。
func normalizePageBasenameForLookup(value string) string {
	base, _ := splitURLSuffix(strings.TrimSpace(decodeImportTarget(value)))
	base = path.Base(base)
	if strings.EqualFold(path.Ext(base), ".md") {
		base = strings.TrimSuffix(base, path.Ext(base))
	}
	base = strings.TrimSpace(base)
	if base == "" || strings.EqualFold(base, "index") {
		return ""
	}
	return strings.ToLower(base)
}

// decodeImportTarget 解码路径中的百分号编码，编码不合法时保留原字符串。
func decodeImportTarget(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

// buildSourceCandidates 按源文件所在目录解释相对目标，根路径目标从工作区根部开始。
// 候选以清理后的原路径开头，再按后缀尝试 Markdown 文件或目录中的 index.md。
// 返回值按产生顺序去重，此处只生成路径字符串，不确认对应文件是否存在。
func buildSourceCandidates(sourcePath string, href string) []string {
	raw := filepath.ToSlash(strings.TrimSpace(href))
	if raw == "" {
		return nil
	}

	var base string
	if strings.HasPrefix(raw, "/") {
		base = path.Clean(strings.TrimPrefix(raw, "/"))
	} else {
		currentDir := path.Dir(filepath.ToSlash(sourcePath))
		if currentDir == "." {
			currentDir = ""
		}
		base = path.Clean(path.Join(currentDir, raw))
	}

	if base == "." || strings.HasPrefix(base, "../") {
		return nil
	}

	candidates := []string{base}
	trimmed := strings.TrimSuffix(base, "/")

	if strings.HasSuffix(raw, "/") {
		candidates = append(candidates, path.Join(trimmed, indexFilename))
	}

	// 扩展名分类不区分大小写，去除 .md 的操作按字面后缀执行。
	ext := strings.ToLower(path.Ext(trimmed))
	switch ext {
	case ".md":
		candidates = append(candidates, path.Join(strings.TrimSuffix(trimmed, ".md"), indexFilename))
	case "":
		candidates = append(candidates, trimmed+".md", path.Join(trimmed, indexFilename))
	}

	return uniqueStrings(candidates)
}

// normalizeSourceCandidateToRoutePath 将文件候选转成页面路由，并去掉末尾 /index。
// 清理后没有页面路径时返回失败，调用方不构造指向根位置的页面链接。
func (t *contentTransformer) normalizeSourceCandidateToRoutePath(candidate string) (string, bool) {
	normalized, ok := t.normalizeWikiHrefToRoutePath(candidate)
	if !ok {
		return "", false
	}

	normalized = strings.TrimSuffix(normalized, "/index")
	normalized = strings.Trim(normalized, "/")
	if normalized == "" {
		return "", false
	}

	return normalized, true
}

// buildTargetPathSuffixKeys 从完整目标路径依次去掉前面的目录，收集所有非空后缀。
// 后缀保留原大小写，例如 a/b/c 会产生 a/b/c、b/c 和 c。
func buildTargetPathSuffixKeys(targetPath string) []string {
	trimmed := strings.Trim(strings.TrimSpace(targetPath), "/")
	if trimmed == "" {
		return nil
	}

	segments := strings.Split(trimmed, "/")
	suffixes := make([]string, 0, len(segments))
	for i := range segments {
		suffixes = append(suffixes, strings.Join(segments[i:], "/"))
	}

	return uniqueStrings(suffixes)
}

// splitMarkdownDestination 对尖括号地址使用第一个右尖括号分界。
// 普通地址在第一个空格或制表符处分开，后半段整体保留，不解析标题的引号。
func splitMarkdownDestination(destination string) (prefix string, href string, suffix string) {
	if destination == "" {
		return "", "", ""
	}
	if destination[0] == '<' {
		if end := strings.IndexByte(destination, '>'); end >= 0 {
			return "<", destination[1:end], destination[end:]
		}
	}

	spaceIdx := strings.IndexAny(destination, " \t")
	if spaceIdx < 0 {
		return "", destination, ""
	}
	return "", destination[:spaceIdx], destination[spaceIdx:]
}

// splitURLSuffix 在最早出现的问号或井号处分开地址，后缀包含分隔符及其后全部文字。
func splitURLSuffix(raw string) (base string, suffix string) {
	queryIdx := strings.IndexByte(raw, '?')
	hashIdx := strings.IndexByte(raw, '#')

	switch {
	case queryIdx >= 0 && hashIdx >= 0:
		cut := queryIdx
		if hashIdx < queryIdx {
			cut = hashIdx
		}
		return raw[:cut], raw[cut:]
	case queryIdx >= 0:
		return raw[:queryIdx], raw[queryIdx:]
	case hashIdx >= 0:
		return raw[:hashIdx], raw[hashIdx:]
	default:
		return raw, ""
	}
}

// normalizeImportedHref 将非盘符形式地址的反斜杠转换为正斜杠。
// 空白目标和形如 C:/ 的盘符路径保持原样，判断时去除的空白不直接写回结果。
func normalizeImportedHref(href string) string {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return href
	}
	if looksLikeWindowsDrivePath(trimmed) {
		return href
	}
	return strings.ReplaceAll(href, `\`, "/")
}

// looksLikeWindowsDrivePath 识别英文字母、冒号、斜杠组成的盘符开头，不检查磁盘。
func looksLikeWindowsDrivePath(value string) bool {
	if len(value) < 3 {
		return false
	}
	drive := value[0]
	return ((drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z')) &&
		value[1] == ':' &&
		(value[2] == '\\' || value[2] == '/')
}

// findMarkdownLinkDestinationEnd 返回目标最外层的右圆括号位置，未闭合返回 -1。
// 内层圆括号逐个累计深度，转义字符和引号不会改变计数规则。
func findMarkdownLinkDestinationEnd(content string, start int) int {
	depth := 0
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// isExternalHref 直接识别 //、mailto: 和 tel: 前缀，其余地址交给 URL 解析。
// URL 解析成功且具有主机名才归为外部地址，单独具有协议名不足以通过该判断。
func isExternalHref(href string) bool {
	if strings.HasPrefix(href, "//") || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
		return true
	}

	u, err := url.Parse(href)
	return err == nil && u.Host != ""
}

// uniqueStrings 去掉空字符串和完全相同的重复项，保留各值第一次出现的顺序。
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// rewriteOutsideCodeSpans 将扫描到的代码区域原样写入，只把间隔中的文字交给回调。
// 围栏和反引号的结束位置由本模块扫描器决定，未闭合区域延伸到全文末尾。
// 回调错误会中止整次转换，已经写入缓冲区的局部结果不会作为返回值交付。
func rewriteOutsideCodeSpans(content string, rewrite func(string) (string, error)) (string, error) {
	var out strings.Builder
	plainStart := 0
	i := 0

	for i < len(content) {
		// 遇到围栏先处理前面的文字，再跳过整个代码块，避免逐字符改写其中链接。
		if fenceLen, fenceChar, ok := detectFenceStart(content, i); ok {
			rewritten, err := rewrite(content[plainStart:i])
			if err != nil {
				return "", err
			}
			out.WriteString(rewritten)

			fenceEnd := findFenceEnd(content, i, fenceChar, fenceLen)
			out.WriteString(content[i:fenceEnd])
			i = fenceEnd
			plainStart = i
			continue
		}

		// 行内反引号采用同样的分段写入方式，代码结束之后继续收集普通文字。
		if content[i] == '`' {
			runLen := countRepeatedByte(content, i, '`')
			rewritten, err := rewrite(content[plainStart:i])
			if err != nil {
				return "", err
			}
			out.WriteString(rewritten)

			codeEnd := findInlineCodeEnd(content, i, runLen)
			out.WriteString(content[i:codeEnd])
			i = codeEnd
			plainStart = i
			continue
		}

		i++
	}

	rewritten, err := rewrite(content[plainStart:])
	if err != nil {
		return "", err
	}
	out.WriteString(rewritten)
	return out.String(), nil
}

// detectFenceStart 只在行首识别围栏，允许最多三个前导空格。
// 至少三个连续反引号或波浪号构成起始围栏，返回实际字符和连续长度。
func detectFenceStart(content string, index int) (int, byte, bool) {
	if !isLineStart(content, index) {
		return 0, 0, false
	}

	lineEnd := index
	for lineEnd < len(content) && content[lineEnd] != '\n' {
		lineEnd++
	}

	line := content[index:lineEnd]
	trimmed := strings.TrimLeft(line, " ")
	indent := len(line) - len(trimmed)
	if indent > 3 || len(trimmed) < 3 {
		return 0, 0, false
	}

	switch trimmed[0] {
	case '`', '~':
		runLen := countLeadingByte(trimmed, trimmed[0])
		if runLen >= 3 {
			return runLen, trimmed[0], true
		}
	}

	return 0, 0, false
}
