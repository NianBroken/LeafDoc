// Package defaultdocs provides the authoritative initial Markdown curriculum.
package defaultdocs

import (
	"embed"
	"fmt"
)

//go:embed content/*.md
var content embed.FS

// StateFile records successful initialization for persistence and snapshots.
const StateFile = "default-documents.json"

// Document describes a page independently of persistence and rendering.
type Document struct {
	Slug    string
	Title   string
	Content string
}

// Load reads the five ordered documents from embedded UTF-8 sources.
func Load() ([]Document, error) {
	documents := []Document{
		{Slug: "markdown-getting-started", Title: "Markdown 入门与写作流程"},
		{Slug: "markdown-text-and-code", Title: "文本格式与代码表达"},
		{Slug: "markdown-lists-and-structure", Title: "列表、引用与内容组织"},
		{Slug: "markdown-links-images-and-tables", Title: "链接、图片与表格"},
		{Slug: "markdown-extensions-and-troubleshooting", Title: "扩展语法、兼容性与排错"},
	}
	for i := range documents {
		body, err := content.ReadFile("content/" + documents[i].Slug + ".md")
		if err != nil {
			return nil, fmt.Errorf("read default document: %w", err)
		}
		documents[i].Content = string(body)
	}
	return documents, nil
}
