// 本文件验证语法树筛选与原文定位的结合，避免把代码样本当作可重写链接。
package links

import (
	"testing"
)

// 相同链接位于行内代码、围栏代码和普通正文，必须只替换普通正文的一个目标。
func TestMarkdownRefactorEngine_Rewrite_SkipsInlineCodeAndCodeBlocks(t *testing.T) {
	content := "`[code](/docs/b)`\n\n```md\n[block](/docs/b)\n```\n\n[real](/docs/b)"

	result := NewMarkdownRefactorEngine().Rewrite(content, "/docs/a", []RewriteRule{{
		OldPath: "/docs/b",
		NewPath: "/guides/b",
	}})

	if result.Count() != 1 {
		t.Fatalf("expected 1 rewrite for real markdown link, got %d", result.Count())
	}
	if result.Content != "`[code](/docs/b)`\n\n```md\n[block](/docs/b)\n```\n\n[real](/guides/b)" {
		t.Fatalf("unexpected rewritten content:\n%s", result.Content)
	}
}

// 引用式链接能形成语法树候选，但没有受支持的行内目标位置，
// 因此不产生替换并报告警告，而不是直接修改引用定义。
func TestMarkdownRefactorEngine_Rewrite_WarnsForReferenceLinks(t *testing.T) {
	content := "[Ref][docs]\n\n[docs]: /docs/b"

	result := NewMarkdownRefactorEngine().Rewrite(content, "/docs/a", []RewriteRule{{
		OldPath: "/docs/b",
		NewPath: "/guides/b",
	}})

	if result.Count() != 0 {
		t.Fatalf("expected no inline rewrite for reference links, got %d", result.Count())
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warning for unsupported reference link syntax")
	}
}
