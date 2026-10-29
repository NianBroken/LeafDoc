package importer

import (
	"net/url"

	"path"
	"path/filepath"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

const indexFilename = "index.md"

type importTarget struct {
	targetPath string
	kind       tree.NodeKind
}

type contentTransformer struct {
	sourceBasePath  string
	assetMaxBytes   int64
	slugger         *tree.SlugService
	pagesBySource   map[string]importTarget
	pagesByBasename map[string][]string
	pagesBySuffix   map[string][]string
	assetUploads    map[string]string
}

// newContentTransformer precomputes source->target lookups from the import plan.
// We resolve links against planned imports so we only rewrite destinations we can actually create.
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

// TransformContent rewrites Markdown links, wiki links, and asset references for one imported page.
// Rewrites only happen outside inline code and fenced blocks so code examples remain untouched.
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

// rewriteMarkdownLinks rewrites regular Markdown links and images in non-code segments only.
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

// rewriteDestination keeps the original Markdown destination wrapper and title suffix intact
// while only replacing the actual href when we can resolve it safely.
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

// resolveDestination first tries to map the href to another imported page.
// If that fails, it falls back to importing a local asset from the source package.
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

// resolvePagePath resolves links only against files that are part of the current import plan.
// This avoids guessing against unrelated existing wiki pages and keeps imports predictable.
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

	// Obsidian also resolves links by note name when that name is unique in the vault.
	// We only apply that fallback for basename-only links within the current import package.
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

func normalizePlanSourcePath(p string) string {
	return strings.ToLower(path.Clean(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(p)), "/")))
}

func basenameOnlyLookupKey(href string) (string, bool) {
	trimmed := strings.TrimSpace(decodeImportTarget(href))
	if trimmed == "" || strings.Contains(trimmed, "/") || strings.HasPrefix(trimmed, ".") {
		return "", false
	}

	key := normalizePageBasenameForLookup(trimmed)
	return key, key != ""
}

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

func decodeImportTarget(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

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

	ext := strings.ToLower(path.Ext(trimmed))
	switch ext {
	case ".md":
		candidates = append(candidates, path.Join(strings.TrimSuffix(trimmed, ".md"), indexFilename))
	case "":
		candidates = append(candidates, trimmed+".md", path.Join(trimmed, indexFilename))
	}

	return uniqueStrings(candidates)
}

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

func looksLikeWindowsDrivePath(value string) bool {
	if len(value) < 3 {
		return false
	}
	drive := value[0]
	return ((drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z')) &&
		value[1] == ':' &&
		(value[2] == '\\' || value[2] == '/')
}

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

func isExternalHref(href string) bool {
	if strings.HasPrefix(href, "//") || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
		return true
	}

	u, err := url.Parse(href)
	return err == nil && u.Host != ""
}

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

// rewriteOutsideCodeSpans applies rewrites only to plain text segments.
// The importer must not rewrite examples inside inline code or fenced code blocks.
func rewriteOutsideCodeSpans(content string, rewrite func(string) (string, error)) (string, error) {
	var out strings.Builder
	plainStart := 0
	i := 0

	for i < len(content) {
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
