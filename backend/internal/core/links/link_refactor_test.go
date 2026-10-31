// 本文件验证普通路径规则对绝对、相对和下级链接的处理，
// 同时确认外部地址、图片和附件目标不作为该入口的页面路径替换对象。
package links

import (
	"strings"
	"testing"
)

// 同一规则改写绝对目标、相对目标和下级目标，并保留下级路径的片段后缀。
func TestMarkdownRefactorEngine_Rewrite_RewritesAbsoluteAndRelativeTargets(t *testing.T) {
	content := `
[Absolute](/docs/b)
[Relative](../b)
[Nested](/docs/b/child#section)
[External](https://example.com/docs/b)
![Image](/docs/b.png)
`

	result := NewMarkdownRefactorEngine().Rewrite(content, "/docs/a", []RewriteRule{{
		OldPath: "/docs/b",
		NewPath: "/guides/b",
	}})

	if result.Count() != 3 {
		t.Fatalf("expected 3 changes, got %d", result.Count())
	}
	if !strings.Contains(result.Content, "[Absolute](/guides/b)") {
		t.Fatalf("expected absolute link rewrite, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "[Relative](../../guides/b)") {
		t.Fatalf("expected relative link rewrite, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "[Nested](/guides/b/child#section)") {
		t.Fatalf("expected subtree link rewrite, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "[External](https://example.com/docs/b)") {
		t.Fatalf("external link should remain unchanged, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "![Image](/docs/b.png)") {
		t.Fatalf("image link should remain unchanged, got:\n%s", result.Content)
	}
}

// 来源和目标都随同一前缀移动时，相对关系保持一致。
// 计划仍包含一条替换，即使计算后的目标文本与原文相同。
func TestMarkdownRefactorEngine_Rewrite_UsesMovedSourcePathForRelativeLinks(t *testing.T) {
	content := `[Relative](../shared)`

	result := NewMarkdownRefactorEngine().Rewrite(content, "/docs/a/page", []RewriteRule{{
		OldPath: "/docs",
		NewPath: "/archive/docs",
	}})

	if result.Count() != 1 {
		t.Fatalf("expected 1 change, got %d", result.Count())
	}
	if result.Content != `[Relative](../shared)` {
		t.Fatalf("expected relative link to be recalculated against moved source path, got %q", result.Content)
	}
}

// 两种附件前缀均保持原文，只重写同时出现的普通页面目标。
func TestMarkdownRefactorEngine_Rewrite_IgnoresAssetLinks(t *testing.T) {
	content := `
[AssetAbs](/assets/abc/manual.pdf)
[AssetRel](assets/abc/manual.pdf)
[Wiki](/docs/b)
`

	result := NewMarkdownRefactorEngine().Rewrite(content, "/docs/a", []RewriteRule{{
		OldPath: "/docs/b",
		NewPath: "/guides/b",
	}})

	if result.Count() != 1 {
		t.Fatalf("expected only wiki link rewrite, got %d", result.Count())
	}
	if !strings.Contains(result.Content, "[AssetAbs](/assets/abc/manual.pdf)") {
		t.Fatalf("absolute asset link should remain unchanged, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "[AssetRel](assets/abc/manual.pdf)") {
		t.Fatalf("relative asset link should remain unchanged, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "[Wiki](/guides/b)") {
		t.Fatalf("wiki link should be rewritten, got:\n%s", result.Content)
	}
}
