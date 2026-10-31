// defaultdocs 包提供程序内置的五篇 Markdown 教程。
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
	// Title 是教程在页面列表中使用的中文标题。
	Title string
	// Content 保存读取后的完整 Markdown 文本，不在本包中解析或渲染。
	Content string
}

// Load 按预先列出的教学顺序读取五篇教程，返回独立的页面描述切片。
// 任何正文读取失败都返回带上下文的错误和 nil，不向调用方交付部分教程。
func Load() ([]Document, error) {
	documents := []Document{
		{Slug: "markdown-getting-started", Title: "Markdown 入门与完整写作流程"},
		{Slug: "markdown-text-and-code", Title: "标题、文本格式与代码"},
		{Slug: "markdown-lists-and-structure", Title: "列表、引用与内容组织"},
		{Slug: "markdown-links-images-and-tables", Title: "链接、图片与表格"},
		{Slug: "markdown-extensions-and-troubleshooting", Title: "扩展语法、实际示例与排错"},
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
