// 本文件解析内置教程，检查页面导航、资源名称和外部链接的允许范围。
package defaultdocs

import (
	"net/url"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/brandassets"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// TestCurriculumRenderedLinksAndResources 检查五篇教程中的非空链接目标。
// Markdown 通过 goldmark 解析为语法树，测试不发起网络请求，也不执行浏览器渲染。
// 图片须对应嵌入资源，站内链接须命中教程路径，外链须使用指定域名上的 HTTPS。
func TestCurriculumRenderedLinksAndResources(t *testing.T) {
	documents, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 5 {
		t.Fatal("expected five tutorials")
	}
	// 先收集全部教程路径，允许任一教程引用列表中尚未遍历到的其他页面。
	routes := map[string]bool{}
	for _, doc := range documents {
		routes["/"+doc.Slug] = true
	}
	for _, doc := range documents {
		source := []byte(doc.Content)
		parsed := goldmark.New().Parser().Parse(text.NewReader(source))
		links := 0
		// 同一语法节点有进入和离开两次回调，只在进入阶段处理一次目标。
		err = ast.Walk(parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			destination := ""
			isImage := false
			switch n := node.(type) {
			case *ast.Link:
				destination = string(n.Destination)
			case *ast.Image:
				destination = string(n.Destination)
				isImage = true
			}
			if destination == "" {
				return ast.WalkContinue, nil
			}
			links++
			if isImage {
				if !strings.HasPrefix(destination, "/defaults/") {
					t.Errorf("%s unknown image %s", doc.Slug, destination)
				} else if _, err := brandassets.Read(strings.TrimPrefix(destination, "/defaults/")); err != nil {
					t.Error(err)
				}
				return ast.WalkContinue, nil
			}
			// 锚点以正文中是否包含对应三级标题文本判断，不计算渲染器生成的标题 ID。
			if strings.HasPrefix(destination, "#") {
				if !strings.Contains(doc.Content, "### "+strings.TrimPrefix(destination, "#")) {
					t.Errorf("%s missing anchor %s", doc.Slug, destination)
				}
				return ast.WalkContinue, nil
			}
			if strings.HasPrefix(destination, "/") {
				if !routes[destination] {
					t.Errorf("%s missing page %s", doc.Slug, destination)
				}
				return ast.WalkContinue, nil
			}
			// 此处验证协议和主机白名单，不验证外部页面此刻能否访问。
			target, err := url.Parse(destination)
			if err != nil || target.Scheme != "https" || (target.Host != "spec.commonmark.org" && target.Host != "github.github.com") {
				t.Errorf("%s unexpected external reference %s", doc.Slug, destination)
			}
			return ast.WalkContinue, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		// 每篇至少包含五个非空目标，保证教程具有实际导航内容。
		if links < 5 {
			t.Errorf("%s incomplete navigation", doc.Slug)
		}
	}
}
