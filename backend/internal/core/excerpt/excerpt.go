// Package excerpt 将 Markdown 正文转换为显示摘要和搜索文本。
// 两种用途共用转换流程，显示摘要执行长度截取，搜索文本保留普通代码块内容。
package excerpt

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/markdown"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// MaxRunes 是截取前的字符数上限，末尾的三个点在截取后追加。
const MaxRunes = 180

// 渲染时允许原始 HTML，转换结果随后交给严格清理策略移除标签。
var mdRenderer = goldmark.New(
	goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
)

var (
	// StrictPolicy 负责 HTML 清理，其他文本格式由后续正则与空白折叠处理。
	sanitize = bluemonday.StrictPolicy()

	// 这些模式覆盖指定写法，不是完整语法解析器。代码段移除模式仅匹配三反引号形式。
	fencedCodePattern    = regexp.MustCompile("(?s)```.*?```")
	mermaidCodePattern   = regexp.MustCompile("(?s)```mermaid.*?```")
	imagePattern         = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	linkPattern          = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	htmlPattern          = regexp.MustCompile(`<[^>]+>`)
	linePrefixPattern    = regexp.MustCompile(`(?m)^\s{0,3}(#{1,6}\s*|[-*+]\s+|\d+\.\s+|>\s?)`)
	wikiImagePattern     = regexp.MustCompile(`!\[\[[^\]\n]*\]\]`)
	wikiLinkAliasPattern = regexp.MustCompile(`\[\[[^\]|\n]+\|([^\]\n]+)\]\]`)
	wikiLinkPattern      = regexp.MustCompile(`\[\[([^\]\n]+)\]\]`)

	shoutoutOpenPattern  = regexp.MustCompile(`^ {0,3}:::\s*(?P<type>[A-Za-z][\w-]*)\s*$`)
	shoutoutClosePattern = regexp.MustCompile(`^ {0,3}:::\s*$`)
	fencePattern         = regexp.MustCompile(`^ {0,3}(?P<marker>` + "`{3,}|~{3,}" + `).*$`)

	// 命名捕获组的位置在包初始化时计算，后续匹配直接按索引读取提示类型和围栏字符。
	shoutoutTypeIdx  = shoutoutOpenPattern.SubexpIndex("type")
	fenceMarkerIdx   = fencePattern.SubexpIndex("marker")
	blockQuotePrefix = regexp.MustCompile(`^(> ?)+`)
)

// fenceState 保存当前代码围栏的字符和开启长度，关闭时要求同类字符且长度不少于它。
type fenceState struct {
	markerChar   byte
	markerLength int
}

// FromContent 尝试剥离页首元信息后生成摘要，解析失败则将完整原文当作正文处理。
// 没有识别到页首元信息时同样保留原文，由 FromBody 完成文本转换和长度截取。
func FromContent(raw string) string {
	_, body, _, err := markdown.ParseFrontmatter(raw)
	if err != nil {
		body = raw
	}
	return FromBody(body)
}

// FromBody 将正文转换为单行文本，空结果直接返回空字符串。
// 超过 MaxRunes 时先按 Unicode 字符截取，再检查靠后空格并追加三个点。
func FromBody(body string) string {
	text := PlainTextFromMarkdown(body)

	if text == "" {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= MaxRunes {
		return text
	}

	truncated := strings.TrimSpace(string(runes[:MaxRunes]))
	// LastIndex 返回字节位置，120 的判断基于字节而非 Unicode 字符数。
	// 满足该条件时在最后一个空格前结束，减少英文单词从中间切断的情况。
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace >= 120 {
		truncated = truncated[:lastSpace]
	}
	return strings.TrimSpace(truncated) + "..."
}

// NormalizeMarkdownBody 将 CRLF 转为 LF，并逐行处理冒号围栏提示块。
// 围栏开始行替换为类型名称，结束行删除，普通行保留；代码围栏内部不作此替换。
// 此函数自身不去掉提示块正文，也不移除所有 Markdown 标记。
func NormalizeMarkdownBody(body string) string {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	output := make([]string, 0, len(lines))
	var outerFence *fenceState

	for _, line := range lines {
		// 已经位于代码块时完整保留该行，仅更新是否遇到匹配关闭围栏。
		if outerFence != nil {
			output = append(output, line)
			outerFence = getFenceState(line, outerFence)
			continue
		}

		if match := shoutoutOpenPattern.FindStringSubmatch(line); match != nil {
			output = append(output, match[shoutoutTypeIdx])
			continue
		}

		if shoutoutClosePattern.MatchString(line) {
			continue
		}

		output = append(output, line)
		outerFence = getFenceState(line, outerFence)
	}

	return strings.Join(output, "\n")
}

// PlainTextFromMarkdown 使用摘要模式转换正文，不执行长度截取。
// 三反引号代码段在渲染前移除，其他形式遵循共享渲染和清理流程。
func PlainTextFromMarkdown(body string) string {
	return plainText(body, false)
}

// PlainTextForSearch 使用搜索模式转换正文，使普通代码块中的词语参与搜索。
// 匹配 Mermaid 专用模式的三反引号代码段仍会移除，不写入图形语法内容。
func PlainTextForSearch(body string) string {
	return plainText(body, true)
}

// applyInlinePatternsOutsideFences 仅在代码块外处理普通图片和双中括号链接。
// 图片保留替代文本，嵌入图片移除，带别名链接保留别名，无别名路径只保留末段。
// 围栏行与代码块内部原样传递，避免搜索时丢失示例中的页面名和文件名。
func applyInlinePatternsOutsideFences(text string) string {
	lines := strings.Split(text, "\n")
	var fence *fenceState
	for i, line := range lines {
		// 检测围栏前移除行首引用标记，供只接受零到三个空格缩进的围栏模式识别。
		// 此处使用临时文本检测，实际返回行仍保留原有引用内容。
		stripped := blockQuotePrefix.ReplaceAllString(line, "")
		nextFence := getFenceState(stripped, fence)
		// 当前和处理后的围栏状态均为空，才属于需要替换的普通正文行。
		// 开启行的后续状态非空，关闭行的当前状态非空，二者均不会进入替换分支。
		if fence == nil && nextFence == nil {
			line = imagePattern.ReplaceAllString(line, "$1")
			line = wikiImagePattern.ReplaceAllString(line, " ")
			line = wikiLinkAliasPattern.ReplaceAllString(line, "$1")
			line = wikiLinkPattern.ReplaceAllStringFunc(line, func(m string) string {
				subs := wikiLinkPattern.FindStringSubmatch(m)
				if len(subs) < 2 {
					return ""
				}
				target := subs[1]
				if idx := strings.LastIndex(target, "/"); idx >= 0 {
					return target[idx+1:]
				}
				return target
			})
			lines[i] = line
		}
		fence = nextFence
	}
	return strings.Join(lines, "\n")
}

// plainText 依次规范提示块与换行、处理代码和链接、渲染 HTML、清理标签并折叠空白。
// 渲染失败时改用规范后的原文本继续清理，不向调用方返回转换错误。
func plainText(body string, includeCode bool) string {
	normalized := NormalizeMarkdownBody(body)
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if !includeCode {
		normalized = fencedCodePattern.ReplaceAllString(normalized, " ")
	} else {
		// 搜索模式仅在此步移除匹配的 Mermaid 段，其他代码交给 Markdown 渲染器保留。
		normalized = mermaidCodePattern.ReplaceAllString(normalized, " ")
	}
	normalized = applyInlinePatternsOutsideFences(normalized)
	var htmlBuf bytes.Buffer
	if err := mdRenderer.Convert([]byte(normalized), &htmlBuf); err != nil {
		htmlBuf.Reset()
		htmlBuf.Write([]byte(normalized))
	}
	// HTML 清理之后继续去除残留标记，最终用 Fields 将各种空白合并为单个空格。
	text := sanitize.Sanitize(htmlBuf.String())
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = fencedCodePattern.ReplaceAllString(text, " ")
	text = imagePattern.ReplaceAllString(text, "$1")
	text = linkPattern.ReplaceAllString(text, "$1")
	text = strings.ReplaceAll(text, "`", "")
	text = htmlPattern.ReplaceAllString(text, " ")
	text = linePrefixPattern.ReplaceAllString(text, "")
	return strings.Join(strings.Fields(text), " ")
}

// getFenceState 根据一行文本返回代码围栏状态，不修改已有状态对象。
// 外部遇到围栏就记录字符与长度，内部仅在同类且不少于开启长度时关闭。
// 模式允许标记后跟任意文字，当前实现不另行校验关闭围栏后的内容。
func getFenceState(line string, currentFence *fenceState) *fenceState {
	match := fencePattern.FindStringSubmatch(line)
	if match == nil {
		return currentFence
	}

	marker := match[fenceMarkerIdx]
	if marker == "" {
		return currentFence
	}

	if currentFence == nil {
		return &fenceState{
			markerChar:   marker[0],
			markerLength: len(marker),
		}
	}

	if marker[0] == currentFence.markerChar && len(marker) >= currentFence.markerLength {
		return nil
	}

	return currentFence
}
