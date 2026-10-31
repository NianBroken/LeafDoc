// 本文件验证链接目标字节区间的替换，以及来源页移动后的相对路径计算。
// 标签、标题和查询后缀属于原文保留部分，不通过整篇 Markdown 重新渲染。
package links

import (
	"strings"
	"testing"
)

// 路径命中规则时，只替换基础路径，查询参数和片段应按原顺序原样拼回。
func TestMarkdownRefactorEngine_Rewrite_KeepsQueryAndFragment(t *testing.T) {
	content := `[Link](/docs/b?mode=1#intro)`

	result := NewMarkdownRefactorEngine().Rewrite(content, "/docs/a", []RewriteRule{{
		OldPath: "/docs/b",
		NewPath: "/guides/b",
	}})

	if result.Count() != 1 {
		t.Fatalf("expected 1 rewrite, got %d", result.Count())
	}
	if result.Content != `[Link](/guides/b?mode=1#intro)` {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

// 目标内的配对圆括号属于文件名，扫描器必须识别嵌套深度而非提前结束目标。
func TestMarkdownRefactorEngine_Rewrite_SupportsParenthesesInDestination(t *testing.T) {
	content := `[Draft](../page_(draft))`

	result := NewMarkdownRefactorEngine().Rewrite(content, "/docs/a", []RewriteRule{{
		OldPath: "/docs/page_(draft)",
		NewPath: "/guides/page_(draft)",
	}})

	if result.Count() != 1 {
		t.Fatalf("expected 1 rewrite, got %d", result.Count())
	}
	if result.Content != `[Draft](../../guides/page_(draft))` {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

// 标签含圆括号且链接带标题时，替换仅覆盖目标地址，不改变标签和标题。
func TestMarkdownRefactorEngine_Rewrite_OnlyChangesDestinationSegment(t *testing.T) {
	content := `[Label with (parens)](/docs/b "Title")`

	result := NewMarkdownRefactorEngine().Rewrite(content, "/docs/a", []RewriteRule{{
		OldPath: "/docs/b",
		NewPath: "/guides/b",
	}})

	if result.Count() != 1 {
		t.Fatalf("expected 1 rewrite, got %d", result.Count())
	}
	if !strings.Contains(result.Content, `[Label with (parens)](/guides/b "Title")`) {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

// 来源跨目录树移动而目标保留原位置时，按页面路径作为目录的规则重算返回路径。
// 期望中的三个上级段与新来源的三个路径段对应。
func TestMarkdownRefactorEngine_RewriteRelativeLinksForPathChange_UsesFolderSemanticsForCrossTreeMove(t *testing.T) {
	content := `[Target](../seite-a)`

	result := NewMarkdownRefactorEngine().RewriteRelativeLinksForPathChange(
		content,
		"/test-link-refactoring/seite-b",
		"/patrick/techtalk/seite-b",
		[]RewriteRule{
			{
				OldPath: "/test-link-refactoring/seite-b",
				NewPath: "/patrick/techtalk/seite-b",
			},
		},
	)

	if result.Count() != 1 {
		t.Fatalf("expected 1 rewrite, got %d", result.Count())
	}
	if result.Content != `[Target](../../../test-link-refactoring/seite-a)` {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

// 附件相对目标使用独立资源地址约定，来源路径变化不触发本入口的目标替换。
func TestMarkdownRefactorEngine_RewriteRelativeLinksForPathChange_IgnoresRelativeAssetLinks(t *testing.T) {
	content := `[Asset](assets/abc/manual.pdf)`

	result := NewMarkdownRefactorEngine().RewriteRelativeLinksForPathChange(
		content,
		"/docs/a",
		"/guides/a",
		[]RewriteRule{
			{
				OldPath: "/docs/a",
				NewPath: "/guides/a",
			},
		},
	)

	if result.Count() != 0 {
		t.Fatalf("expected no rewrite for asset link, got %d", result.Count())
	}
	if result.Content != content {
		t.Fatalf("asset link should remain unchanged, got %q", result.Content)
	}
}
