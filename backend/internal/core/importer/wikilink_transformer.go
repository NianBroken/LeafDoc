package importer

import (
	"path"

	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// rewriteWikiLinks 扫描双中括号链接和带感叹号的嵌入标记，逐个解析目标与显示文字。
// 附件和带查询串或片段的页面目标输出 Markdown 链接；其他页面保留双中括号形式。
// 找不到闭合标记时保留余下正文，附件上传错误则中止转换并返回错误。
func (t *contentTransformer) rewriteWikiLinks(
	userID string,
	sourcePath string,
	page *tree.Page,
	content string,
	wiki ImporterWiki,
) (string, error) {
	var out strings.Builder
	for i := 0; i < len(content); {
		nextImage := strings.Index(content[i:], "![[")
		nextLink := strings.Index(content[i:], "[[")
		if nextImage < 0 && nextLink < 0 {
			out.WriteString(content[i:])
			break
		}

		// 两种起始标记中选择最靠前的一处，嵌入标记包含的 [[ 不会被单独处理。
		next := -1
		isImage := false
		switch {
		case nextImage >= 0 && (nextLink < 0 || nextImage <= nextLink):
			next = i + nextImage
			isImage = true
		case nextLink >= 0:
			next = i + nextLink
		}

		out.WriteString(content[i:next])

		startOffset := 2
		if isImage {
			startOffset = 3
		}
		end := strings.Index(content[next+startOffset:], "]]")
		if end < 0 {
			out.WriteString(content[next:])
			break
		}
		end += next + startOffset

		inner := strings.TrimSpace(content[next+startOffset : end])
		targetPart, label := splitWikiLink(inner)
		targetPart = normalizeImportedHref(targetPart)
		_, anchorSuffix := splitURLSuffix(targetPart)
		href, isAsset, err := t.resolveDestination(userID, sourcePath, page, targetPart, wiki)
		if err != nil {
			return "", err
		}

		// 附件使用上传得到的公开路径，嵌入标记或图片扩展名会输出图片语法。
		if isAsset {
			if label == "" {
				label = defaultWikiLinkLabel(targetPart)
			}
			if shouldRenderWikiLinkAsImage(isImage, targetPart) {
				out.WriteByte('!')
			}
			out.WriteString("[")
			out.WriteString(label)
			out.WriteString("](")
			out.WriteString(href)
			out.WriteByte(')')
			i = end + 2
			continue
		}

		// 查询串与片段需要保留在地址中，因此使用 Markdown 链接承载整个后缀。
		// 页面未在计划中命中时尝试构造路由，无法构造则保留原链接文字。
		if anchorSuffix != "" {
			if href == "" {
				fallbackHref, ok := t.fallbackWikiPageHref(sourcePath, targetPart)
				if !ok {
					out.WriteString(content[next : end+2])
					i = end + 2
					continue
				}
				href = fallbackHref
			}
			if label == "" {
				label = defaultWikiLinkLabel(targetPart)
			}
			out.WriteString("[")
			out.WriteString(label)
			out.WriteString("](")
			out.WriteString(href)
			out.WriteByte(')')
			i = end + 2
			continue
		}

		// 不带后缀的页面继续输出双中括号语法，交给正文的页面链接解析流程。
		// 带斜杠的目标优先使用计划解析路径，否则逐段生成有效路径段。
		// 页面嵌入标记在这条分支不保留感叹号，输出的是普通页面引用。
		target := targetPart
		if strings.Contains(targetPart, "/") {
			if href != "" {
				target = strings.TrimPrefix(href, "/")
			} else if slugged, ok := t.normalizeWikiHrefToRoutePath(targetPart); ok {
				target = slugged
			}
		}
		out.WriteString("[[")
		if label != "" {
			out.WriteString(target)
			out.WriteByte('|')
			out.WriteString(label)
		} else {
			out.WriteString(target)
		}
		out.WriteString("]]")
		i = end + 2
	}

	return out.String(), nil
}

// fallbackWikiPageHref 在没有页面命中结果时，为带后缀的页面引用构造路由。
// 外部地址、空目标、非 Markdown 附件和具有多个同名候选的目标被排除。
// 点或斜杠开头的地址采用首个源路径候选，其余地址直接逐段转换；不创建页面。
func (t *contentTransformer) fallbackWikiPageHref(sourcePath string, href string) (string, bool) {
	rawTarget, suffix := splitURLSuffix(href)
	if rawTarget == "" || isExternalHref(rawTarget) || strings.HasPrefix(rawTarget, "#") {
		return "", false
	}

	if isNonMarkdownAssetTarget(rawTarget) {
		return "", false
	}

	if basenameKey, ok := basenameOnlyLookupKey(rawTarget); ok {
		if matches := uniqueStrings(t.pagesByBasename[basenameKey]); len(matches) > 1 {
			return "", false
		}
	}

	if strings.HasPrefix(rawTarget, ".") || strings.HasPrefix(rawTarget, "/") {
		candidates := buildSourceCandidates(sourcePath, rawTarget)
		if len(candidates) == 0 {
			return "", false
		}
		routePath, ok := t.normalizeSourceCandidateToRoutePath(candidates[0])
		if !ok {
			return "", false
		}
		return "/" + routePath + suffix, true
	}

	routePath, ok := t.normalizeWikiHrefToRoutePath(rawTarget)
	if !ok {
		return "", false
	}
	return "/" + routePath + suffix, true
}

// normalizeWikiHrefToRoutePath 解码目标，去掉两端各一个斜杠，再转换每个路径段。
// 末段的 Markdown 扩展名被去除；任意空段或不能生成路径段的名称都会使转换失败。
func (t *contentTransformer) normalizeWikiHrefToRoutePath(href string) (string, bool) {
	decoded := strings.TrimSpace(decodeImportTarget(href))
	if decoded == "" {
		return "", false
	}

	decoded = strings.TrimPrefix(decoded, "/")
	decoded = strings.TrimSuffix(decoded, "/")
	if decoded == "" {
		return "", false
	}

	segments := strings.Split(decoded, "/")
	for i, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return "", false
		}

		if i == len(segments)-1 && strings.EqualFold(path.Ext(segment), ".md") {
			segment = strings.TrimSuffix(segment, path.Ext(segment))
		}

		safe := t.slugger.GenerateValidSlug(segment)
		if safe == "" {
			return "", false
		}
		segments[i] = safe
	}

	return strings.Join(segments, "/"), true
}

// splitWikiLink 只在第一个竖线处分开目标和显示文字，两部分分别去除两端空白。
// 后续竖线保留在显示文字中，没有竖线时显示文字为空。
func splitWikiLink(inner string) (target string, label string) {
	parts := strings.SplitN(inner, "|", 2)
	target = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		label = strings.TrimSpace(parts[1])
	}
	return target, label
}

// defaultWikiLinkLabel 取目标末段作为默认显示文字，去掉后缀、末尾斜杠和字面 .md。
// 结果为空时直接返回原目标，避免生成空标签。
func defaultWikiLinkLabel(target string) string {
	base, _ := splitURLSuffix(target)
	base = strings.TrimSuffix(base, "/")
	base = strings.TrimSuffix(base, ".md")
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	if idx := strings.LastIndex(base, "#"); idx >= 0 {
		base = base[:idx]
	}
	if base == "" {
		return target
	}
	return base
}

// shouldRenderWikiLinkAsImage 对嵌入标记或具有已识别图片扩展名的附件输出图片语法。
func shouldRenderWikiLinkAsImage(isEmbed bool, target string) bool {
	return isEmbed || isImageAssetTarget(target)
}
