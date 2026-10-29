package defaultdocs

import (
	"github.com/NianBroken/LeafDoc/internal/core/brandassets"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"net/url"
	"strings"
	"testing"
)

// Every rendered document link resolves to another seeded page, a local asset,
// an explicit heading, or one of the verified canonical specification URLs.
func TestCurriculumRenderedLinksAndResources(t *testing.T) {
	documents, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 5 {
		t.Fatal("expected five tutorials")
	}
	routes := map[string]bool{}
	for _, doc := range documents {
		routes["/"+doc.Slug] = true
	}
	for _, doc := range documents {
		source := []byte(doc.Content)
		parsed := goldmark.New().Parser().Parse(text.NewReader(source))
		links := 0
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
			target, err := url.Parse(destination)
			if err != nil || target.Scheme != "https" || (target.Host != "spec.commonmark.org" && target.Host != "github.github.com") {
				t.Errorf("%s unexpected external reference %s", doc.Slug, destination)
			}
			return ast.WalkContinue, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if links < 5 {
			t.Errorf("%s incomplete navigation", doc.Slug)
		}
	}
}
