package importer

import (
	"path"

	"strings"

	"github.com/perber/wiki/internal/core/tree"
)

// rewriteWikiLinks handles Obsidian-style wiki links and converts them to plain Markdown links.
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

		// Native WikiLinks don't support anchors — convert anchor links to Markdown.
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

		// Plain page WikiLink (no anchor, no asset): keep as [[...]] so LeafWiki's
		// native title-based resolution handles it — rename-aware and self-healing.
		//
		// Path-hinted links ([[Folder/Page]]) use the plan's resolved slug path so the
		// refactoring engine's path-hint regex matches exactly on rename. Unresolved
		// path-hinted links are slugified so the link indexer and heal mechanism can
		// match them when the target page is later created.
		// Obsidian page-embeds (![[Page]]) are kept as plain [[...]] because LeafWiki
		// has no page-transclusion support.
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

func splitWikiLink(inner string) (target string, label string) {
	parts := strings.SplitN(inner, "|", 2)
	target = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		label = strings.TrimSpace(parts[1])
	}
	return target, label
}

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

func shouldRenderWikiLinkAsImage(isEmbed bool, target string) bool {
	return isEmbed || isImageAssetTarget(target)
}
