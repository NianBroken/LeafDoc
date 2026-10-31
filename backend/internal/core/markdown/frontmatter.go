// 本文件解析和组装 Markdown 页首的 YAML 元信息，区分系统字段与自定义字段。
// 解析容错仅作用于内存文本，只有调用方执行保存时才会写入实际存储。
package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	yaml "gopkg.in/yaml.v3"
)

// invalidYAMLKeyRune 为页首结构识别限定键名字符，不是完整 YAML 键名校验器。
// 字母、数字、下划线和连字符以外的字符均返回 true。
func invalidYAMLKeyRune(r rune) bool {
	//nolint:staticcheck
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-')
}

// Frontmatter 保存 LeafDoc 系统字段和其余 YAML 字段，供页面读取与正文组装使用。
// 时间字段保存字符串，解析时不在此结构中验证日期是否符合业务要求。
type Frontmatter struct {
	// 系统字段的 YAML 名称带 leafdoc_ 前缀，JSON 名称用于页面元信息表示。
	LeafDocID           string `yaml:"leafdoc_id,omitempty" json:"id,omitempty"`
	LeafDocTitle        string `yaml:"leafdoc_title,omitempty" json:"title,omitempty"`
	LeafDocCreatedAt    string `yaml:"leafdoc_created_at,omitempty" json:"createdAt,omitempty"`
	LeafDocUpdatedAt    string `yaml:"leafdoc_updated_at,omitempty" json:"updatedAt,omitempty"`
	LeafDocCreatorID    string `yaml:"leafdoc_creator_id,omitempty" json:"creatorId,omitempty"`
	LeafDocLastAuthorID string `yaml:"leafdoc_last_author_id,omitempty" json:"lastAuthorId,omitempty"`
	LeafDocPinned       bool   `yaml:"leafdoc_pinned,omitempty" json:"pinned,omitempty"`
	// ExtraFields 由手动解析和组装处理，不参加此结构的直接 YAML 或 JSON 编码。
	ExtraFields map[string]interface{} `yaml:"-" json:"-"`

	// repaired 表示首次 YAML 解析失败后，内存文本经规范处理并重试成功。
	// 该标记不参与序列化，也不表示原文件已经被写回。
	repaired bool
}

// WasRepaired 返回本次解析是否通过容错重试完成，供调用方决定是否记录提示。
// 首次解析直接成功时为 false，不根据字段内容推断是否需要重新保存。
func (fm Frontmatter) WasRepaired() bool {
	return fm.repaired
}

var unquotedTemplatePlaceholderLine = regexp.MustCompile(`^(\s*[^:\n]+:\s*)(\{\{[^}\n]+\}\})(\s*(?:#.*)?)$`)

// parseFrontmatterYAML 将 YAML 映射拆为系统字段与附加字段。
// 首次失败时只在规范处理确实改变文本后重试，重试失败返回最初的解析错误。
func parseFrontmatterYAML(yamlPart string) (Frontmatter, error) {
	var raw map[string]interface{}
	repaired := false
	if err := yaml.Unmarshal([]byte(yamlPart), &raw); err != nil {
		sanitized, changed := sanitizeFrontmatterYAML(yamlPart)
		if !changed {
			return Frontmatter{}, errors.Join(ErrFrontmatterParse, err)
		}
		if retryErr := yaml.Unmarshal([]byte(sanitized), &raw); retryErr != nil {
			return Frontmatter{}, errors.Join(ErrFrontmatterParse, err)
		}
		repaired = true
	}
	if raw == nil {
		raw = map[string]interface{}{}
	}

	fm := Frontmatter{ExtraFields: map[string]interface{}{}, repaired: repaired}

	if value, ok := raw["leafdoc_id"]; ok {
		fm.LeafDocID = fm.stripSingleAndDoubleQuotes(strings.TrimSpace(valueToString(value)))
	}

	// 专用标题键优先，即使其值为空也不会再回退到普通 title 键。
	if value, ok := raw["leafdoc_title"]; ok {
		fm.LeafDocTitle = fm.stripSingleAndDoubleQuotes(valueToString(value))
	} else if value, ok := raw["title"]; ok {
		fm.LeafDocTitle = fm.stripSingleAndDoubleQuotes(valueToString(value))
	}
	if value, ok := raw["leafdoc_created_at"]; ok {
		fm.LeafDocCreatedAt = fm.stripSingleAndDoubleQuotes(strings.TrimSpace(valueToString(value)))
	}
	if value, ok := raw["leafdoc_updated_at"]; ok {
		fm.LeafDocUpdatedAt = fm.stripSingleAndDoubleQuotes(strings.TrimSpace(valueToString(value)))
	}
	if value, ok := raw["leafdoc_creator_id"]; ok {
		fm.LeafDocCreatorID = fm.stripSingleAndDoubleQuotes(strings.TrimSpace(valueToString(value)))
	}
	if value, ok := raw["leafdoc_last_author_id"]; ok {
		fm.LeafDocLastAuthorID = fm.stripSingleAndDoubleQuotes(strings.TrimSpace(valueToString(value)))
	}
	// 置顶状态只接受 YAML 布尔值，字符串形式的 true 不会转为 true。
	if value, ok := raw["leafdoc_pinned"]; ok {
		if b, ok := value.(bool); ok {
			fm.LeafDocPinned = b
		}
	}

	// 仅排除这里明确列出的系统键，普通 title 与其他未列出的键仍保留在附加字段中。
	for key, value := range raw {
		switch key {
		case "leafdoc_id", "leafdoc_title", "leafdoc_created_at", "leafdoc_updated_at", "leafdoc_creator_id", "leafdoc_last_author_id", "leafdoc_pinned":
			continue
		default:
			fm.ExtraFields[key] = value
		}
	}

	if len(fm.ExtraFields) == 0 {
		fm.ExtraFields = nil
	}

	return fm, nil
}

// sanitizeLines 按换行拆分文本，将每行交给 fn 判断是否替换。
// 回调返回 true 才写回对应位置，最终按原换行分隔方式拼接并汇总是否发生替换。
func sanitizeLines(yamlPart string, fn func(line string) (string, bool)) (string, bool) {
	lines := strings.Split(yamlPart, "\n")
	changed := false

	for i, line := range lines {
		if rewritten, ok := fn(line); ok {
			lines[i] = rewritten
			changed = true
		}
	}

	return strings.Join(lines, "\n"), changed
}

// sanitizeTemplatePlaceholderFrontmatter 识别值仅为双花括号模板的匹配行。
// 模板内容用 strconv.Quote 包成字符串，键前缀和尾部空白或注释保留。
func sanitizeTemplatePlaceholderFrontmatter(yamlPart string) (string, bool) {
	return sanitizeLines(yamlPart, func(line string) (string, bool) {
		matches := unquotedTemplatePlaceholderLine.FindStringSubmatch(line)
		if matches == nil {
			return "", false
		}
		return matches[1] + strconv.Quote(matches[2]) + matches[3], true
	})
}

// knownLeafDocStringFrontmatterKeys 列出容错处理允许加引号的六个系统字符串键。
// 布尔键 leafdoc_pinned 不在其中，避免将布尔值转成字符串后失去类型含义。
var knownLeafDocStringFrontmatterKeys = []string{
	"leafdoc_id",
	"leafdoc_title",
	"leafdoc_created_at",
	"leafdoc_updated_at",
	"leafdoc_creator_id",
	"leafdoc_last_author_id",
}

// unquotedLeafDocStringFieldLine 只匹配从行首开始的已知系统字符串键。
// 不接受键名前的缩进，从而跳过其他字段的缩进多行正文；值必须从非空白字符开始。
var unquotedLeafDocStringFieldLine = regexp.MustCompile(
	`^((?:` + strings.Join(knownLeafDocStringFrontmatterKeys, "|") + `)\s*:\s*)(\S.*)$`,
)

// lineIsValidYAML 将单行直接解析为映射，只以 YAML 解析是否报错作为结果。
// 它不核验字段名、业务类型或该行在完整文档中的上下文。
func lineIsValidYAML(line string) bool {
	var probe map[string]interface{}
	return yaml.Unmarshal([]byte(line), &probe) == nil
}

// sanitizeUnquotedColonInLeafDocStringFields 处理已知系统字符串键的非法单行值。
// 值以显式集合、锚点、别名、标签、多行文本或指令标记开头时保留原状。
// 其余候选行先经真实 YAML 解析器验证，仅对解析失败的值整体加引号。
// 普通用户字段不参与此处理，已合法的引号和行尾 YAML 注释也不会被重写。
func sanitizeUnquotedColonInLeafDocStringFields(yamlPart string) (string, bool) {
	return sanitizeLines(yamlPart, func(line string) (string, bool) {
		matches := unquotedLeafDocStringFieldLine.FindStringSubmatch(line)
		if matches == nil {
			return "", false
		}

		prefix, value := matches[1], strings.TrimRight(matches[2], " \t")
		if value == "" {
			return "", false
		}
		// 显式 YAML 结构不强制转换成普通文本，保留其原有解析成功或失败结果。
		// 双花括号不按单花括号集合排除，让模板组合值继续进入真实解析与加引号判断。
		if strings.IndexByte("[&*!|>%", value[0]) >= 0 {
			return "", false
		}
		if value[0] == '{' && !strings.HasPrefix(value, "{{") {
			return "", false
		}
		if lineIsValidYAML(prefix + value) {
			return "", false
		}

		return prefix + strconv.Quote(value), true
	})
}

// sanitizeFrontmatterYAML 顺序执行模板值处理和系统字符串值处理。
// 第二步读取第一步结果，并重新解析候选行，因此已经合法的行不会再次加引号。
// 返回的布尔值表示至少一个步骤发生替换，不表示整个 YAML 必定解析成功。
func sanitizeFrontmatterYAML(yamlPart string) (string, bool) {
	out := yamlPart
	changedAny := false

	if sanitized, changed := sanitizeTemplatePlaceholderFrontmatter(out); changed {
		out, changedAny = sanitized, true
	}
	if sanitized, changed := sanitizeUnquotedColonInLeafDocStringFields(out); changed {
		out, changedAny = sanitized, true
	}

	return out, changedAny
}

// valueToString 将 nil 转为空串，字符串原样返回，时间统一按 UTC 的 RFC3339 输出。
// 其他类型通过 fmt.Sprint 表示，不拒绝集合或其他非字符串值。
func valueToString(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case time.Time:
		return typed.UTC().Format(time.RFC3339)
	default:
		return fmt.Sprint(typed)
	}
}

// stripSingleAndDoubleQuotes 依次去掉两端的双引号字符和单引号字符。
// 这里不执行字符串反转义，也不检查两端引号是否成对。
func (fm *Frontmatter) stripSingleAndDoubleQuotes(s string) string {
	s = strings.Trim(s, `"`)
	s = strings.Trim(s, `'`)
	return s
}

// splitFrontmatter 在首行围栏和结束围栏之间识别至少一条形似 YAML 键值的内容。
// 没有足够结构依据时返回原始 md 作为正文，不把普通分隔线误作页首元信息。
func splitFrontmatter(md string) (yamlPart string, body string, has bool) {
	// 识别使用去掉开头 BOM 并统一换行的临时副本，未识别成功时仍返回原文。
	s := strings.TrimPrefix(md, "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// 开启标记必须位于首行且精确为三个连字符，前后附加空白不会通过此判断。
	if s != "---" && !strings.HasPrefix(s, "---\n") {
		return "", md, false
	}

	// 定位开启行的换行，没有换行的单个标记不构成完整元信息块。
	firstNL := strings.IndexByte(s, '\n')
	if firstNL == -1 {
		// 只有开启标记而没有后续内容，保留完整原文。
		return "", md, false
	}
	if strings.TrimSpace(s[:firstNL]) != "---" {
		return "", md, false
	}

	// 逐行查找结束标记，结束行比较前会去掉两端空白，也接受位于文件结尾的标记。
	pos := firstNL + 1
	yamlStart := pos

	endDelimLineStart := -1
	endDelimLineEnd := -1

	looksLikeYAML := false

	for pos <= len(s) {
		nextNL := strings.IndexByte(s[pos:], '\n')
		var line string
		var lineEnd int
		if nextNL == -1 {
			lineEnd = len(s)
			line = s[pos:lineEnd]
		} else {
			lineEnd = pos + nextNL
			line = s[pos:lineEnd]
		}

		trim := strings.TrimSpace(line)
		if trim == "---" {
			endDelimLineStart = pos
			endDelimLineEnd = lineEnd
			break
		}

		// 排除空行和注释行，冒号前需为允许的键名，冒号后需为空或紧接空白。
		if trim != "" && !strings.HasPrefix(trim, "#") {
			if idx := strings.IndexByte(trim, ':'); idx > 0 {
				key := strings.TrimSpace(trim[:idx])
				hasYAMLSeparator := idx == len(trim)-1 || unicode.IsSpace(rune(trim[idx+1]))
				if hasYAMLSeparator && key != "" && strings.IndexFunc(key, invalidYAMLKeyRune) == -1 {
					looksLikeYAML = true
				}
			}
		}

		if nextNL == -1 {
			pos = len(s) + 1
		} else {
			pos = lineEnd + 1
		}
	}

	if endDelimLineStart == -1 {
		return "", md, false
	}

	if !looksLikeYAML {
		return "", md, false
	}

	yamlPart = s[yamlStart:endDelimLineStart]
	yamlPart = strings.TrimSuffix(yamlPart, "\n")

	bodyStart := endDelimLineEnd
	if bodyStart < len(s) && s[bodyStart:bodyStart+1] == "\n" {
		bodyStart++
	}
	body = s[bodyStart:]

	return yamlPart, body, true
}

// ParseFrontmatter 在内存中分离并解析元信息，不执行路径标题回退或文件写回。
// 未识别到结构时 has 为 false 且正文为原文；已识别但 YAML 失败时 has 为 true，正文仍为原文。
// 仅解析成功才返回分隔后的正文，错误信息保留 YAML 分类与底层解析原因。
func ParseFrontmatter(md string) (fm Frontmatter, body string, has bool, err error) {
	yamlPart, body, has := splitFrontmatter(md)
	if !has {
		return Frontmatter{}, md, false, nil
	}

	fm, err = parseFrontmatterYAML(yamlPart)
	if err != nil {
		return Frontmatter{}, md, true, err
	}
	return fm, body, true, nil
}

// toYAMLNode 使用 yaml.Node.Encode 保留值的 YAML 类型，编码失败直接返回错误。
func toYAMLNode(value interface{}) (*yaml.Node, error) {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, err
	}
	return &node, nil
}

// buildExtraFieldsMapping 按键的字典序组织映射节点，使相同字段获得稳定输出顺序。
// 键显式编码为字符串，值交给 YAML 库编码，任一字段失败即放弃整个映射。
func buildExtraFieldsMapping(extraFields map[string]interface{}) (*yaml.Node, error) {
	mapping := &yaml.Node{Kind: yaml.MappingNode}

	extraKeys := make([]string, 0, len(extraFields))
	for key := range extraFields {
		extraKeys = append(extraKeys, key)
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		valueNode, err := toYAMLNode(extraFields[key])
		if err != nil {
			return nil, err
		}
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			valueNode,
		)
	}

	return mapping, nil
}

// buildMarkdownFromMapping 为非空映射输出首尾围栏并接上正文，正文内容原样保留。
// nil 或没有字段的映射直接返回正文，不输出空元信息块。
func buildMarkdownFromMapping(mapping *yaml.Node, body string) (string, error) {
	if mapping == nil || len(mapping.Content) == 0 {
		return body, nil
	}

	b, err := yaml.Marshal(mapping)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(b)
	out.WriteString("---\n")
	out.WriteString(body)
	return out.String(), nil
}

// BuildMarkdownWithExtraFrontmatter 仅将附加字段编码为元信息，不自动注入系统字段。
// 空字段集合直接返回正文，字段排序与类型编码复用内部映射构建逻辑。
func BuildMarkdownWithExtraFrontmatter(extraFields map[string]interface{}, body string) (string, error) {
	if len(extraFields) == 0 {
		return body, nil
	}

	mapping, err := buildExtraFieldsMapping(extraFields)
	if err != nil {
		return "", err
	}

	return buildMarkdownFromMapping(mapping, body)
}

// BuildMarkdownWithFrontmatter 先输出排序后的附加字段，再按固定顺序追加系统字段。
// ID 去掉空白后为空时只输出附加字段，其他系统值均不参与本次组装。
// 有 ID 时非空字符串系统字段去掉两端空白后输出，置顶仅在 true 时输出。
// 本函数不从 ExtraFields 中剔除与系统字段重名的键，调用方需维持字段分离。
func BuildMarkdownWithFrontmatter(fm Frontmatter, body string) (string, error) {
	if strings.TrimSpace(fm.LeafDocID) == "" {
		return BuildMarkdownWithExtraFrontmatter(fm.ExtraFields, body)
	}

	mapping, err := buildExtraFieldsMapping(fm.ExtraFields)
	if err != nil {
		return "", err
	}

	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "leafdoc_id"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.TrimSpace(fm.LeafDocID)},
	)
	if strings.TrimSpace(fm.LeafDocTitle) != "" {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "leafdoc_title"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.TrimSpace(fm.LeafDocTitle)},
		)
	}
	if strings.TrimSpace(fm.LeafDocCreatedAt) != "" {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "leafdoc_created_at"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.TrimSpace(fm.LeafDocCreatedAt)},
		)
	}
	if strings.TrimSpace(fm.LeafDocUpdatedAt) != "" {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "leafdoc_updated_at"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.TrimSpace(fm.LeafDocUpdatedAt)},
		)
	}
	if strings.TrimSpace(fm.LeafDocCreatorID) != "" {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "leafdoc_creator_id"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.TrimSpace(fm.LeafDocCreatorID)},
		)
	}
	if strings.TrimSpace(fm.LeafDocLastAuthorID) != "" {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "leafdoc_last_author_id"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.TrimSpace(fm.LeafDocLastAuthorID)},
		)
	}
	if fm.LeafDocPinned {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "leafdoc_pinned"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		)
	}

	return buildMarkdownFromMapping(mapping, body)
}
