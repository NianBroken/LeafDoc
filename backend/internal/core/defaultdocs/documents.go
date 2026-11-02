// defaultdocs 包提供程序内置的十篇 Markdown 教程。
// 本包读取嵌入正文并组装页面描述，页面持久化和初始化状态判断由调用方负责。
package defaultdocs

import (
	"embed"
	"fmt"
)

// content 保存构建时 content 目录中的 Markdown 原文，运行时无需读取外部教程文件。
// 下方的 go:embed 指令负责收集正文，Load 的明确列表决定对外返回哪些教程。
//
//go:embed content/*.md
var content embed.FS

// StateFile 是教程初始化状态使用的文件名，供持久化和快照相关代码共同引用。
// 声明常量本身不会创建状态文件，也不会将初始化标记为成功。
const StateFile = "default-documents.json"

// Document 将页面路径标识、显示标题与 Markdown 原文放在同一个描述中。
// 它不包含数据库标识、父子节点或渲染结果，调用方据此建立实际页面。
type Document struct {
	// Slug 用于定位对应的嵌入正文，同时提供稳定的页面路径标识。
	Slug string
	// ParentSlug 指向先于本页面声明的章节，为空表示根页面。
	ParentSlug string
	// Path 是完整页面路径，由父子关系计算。
	Path string
	// Title 是教程在页面列表中使用的中文标题。
	Title string
	// Content 保存读取后的完整 Markdown 文本，不在本包中解析或渲染。
	Content string
}

// Load 按预先列出的教学顺序读取十篇教程，返回独立的页面描述切片。
// 任何正文读取失败都返回带上下文的错误和 nil，不向调用方交付部分教程。
func Load() ([]Document, error) {
	documents := []Document{
		{Slug: "markdown-start", Title: "Markdown 入门与编辑实践", ParentSlug: ""},
		{Slug: "headings", Title: "标题、段落与换行", ParentSlug: "markdown-start"},
		{Slug: "formatting", Title: "强调、转义与行内代码", ParentSlug: "markdown-start"},
		{Slug: "markdown-structure", Title: "Markdown 内容组织", ParentSlug: ""},
		{Slug: "lists", Title: "列表与任务清单", ParentSlug: "markdown-structure"},
		{Slug: "quotes-code", Title: "引用与代码块", ParentSlug: "markdown-structure"},
		{Slug: "tables", Title: "表格与对齐", ParentSlug: "markdown-structure"},
		{Slug: "markdown-practice", Title: "Markdown 链接与扩展实践", ParentSlug: ""},
		{Slug: "links-assets", Title: "页面链接、图片与附件", ParentSlug: "markdown-practice"},
		{Slug: "extensions", Title: "数学公式、Mermaid 与排错", ParentSlug: "markdown-practice"},
	}
	// 父页面必须先出现，路径唯一且层级不会形成循环。
	paths := map[string]string{}
	for i := range documents {
		d := &documents[i]
		if _, exists := paths[d.Slug]; exists {
			return nil, fmt.Errorf("duplicate default document: %s", d.Slug)
		}
		d.Path = "/" + d.Slug
		if d.ParentSlug != "" {
			parent, exists := paths[d.ParentSlug]
			if !exists {
				return nil, fmt.Errorf("missing preceding parent: %s", d.ParentSlug)
			}
			d.Path = parent + d.Path
		}
		paths[d.Slug] = d.Path
	}

	// 按切片下标写回正文，保持返回顺序与上方页面列表一致。
	for i := range documents {
		body, err := content.ReadFile("content/" + documents[i].Slug + ".md")
		if err != nil {
			return nil, fmt.Errorf("read default document: %w", err)
		}
		// 字节直接转换为字符串，不改写换行、语法、链接或教程内容。
		documents[i].Content = string(body)
	}
	return documents, nil
}
