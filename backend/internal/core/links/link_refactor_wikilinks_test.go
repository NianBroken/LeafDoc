// 本文件验证双链标题、路径提示和预览查找。所有正文与期望值均为测试数据，
// 测试关注保留别名、排除代码、大小写规则及重叠匹配的确定优先级。
package links

import (
	"testing"
)

// 下列用例检查实际重写结果和替换条目数量。

// 标题规则同时处理普通双链和带别名双链，别名文字保持原样。
func TestRewriteWikiLinks_TitleRename(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "See [[Project Plan]] and [[Project Plan|our plan]] for details."

	result := engine.RewriteWikiLinks(content, []RewriteRule{{
		OldPath:  "/docs/project-plan",
		NewPath:  "/docs/project-overview",
		OldTitle: "Project Plan",
		NewTitle: "Project Overview",
	}})

	want := "See [[Project Overview]] and [[Project Overview|our plan]] for details."
	if result.Content != want {
		t.Errorf("got %q, want %q", result.Content, want)
	}
	if result.Count() != 2 {
		t.Errorf("expected 2 replacements, got %d", result.Count())
	}
}

// 标题匹配忽略大小写并接受边缘空白，输出采用规则目标文字且移除边缘空白。
func TestRewriteWikiLinks_TitleRenameCaseInsensitive(t *testing.T) {
	engine := NewMarkdownRefactorEngine()

	cases := []struct{ input, want string }{
		{"See [[project plan]] here.", "See [[Project Overview]] here."},
		{"See [[ project plan ]] here.", "See [[Project Overview]] here."},
		{"See [[ Project Plan  ]] here.", "See [[Project Overview]] here."},
	}
	for _, tc := range cases {
		result := engine.RewriteWikiLinks(tc.input, []RewriteRule{{
			OldTitle: "Project Plan",
			NewTitle: "Project Overview",
		}})
		if result.Content != tc.want {
			t.Errorf("input %q: got %q, want %q", tc.input, result.Content, tc.want)
		}
	}
}

// 路径提示使用去掉开头斜杠的路径文本，普通形式和别名形式均替换目标部分。
func TestRewriteWikiLinks_PathHintRewrite(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "See [[docs/intro]] and [[docs/intro|Introduction]]."

	result := engine.RewriteWikiLinks(content, []RewriteRule{{
		OldPath: "/docs/intro",
		NewPath: "/guides/intro",
	}})

	want := "See [[guides/intro]] and [[guides/intro|Introduction]]."
	if result.Content != want {
		t.Errorf("got %q, want %q", result.Content, want)
	}
	if result.Count() != 2 {
		t.Errorf("expected 2 replacements, got %d", result.Count())
	}
}

// 同一标题出现在行内代码、围栏代码和普通正文中时，只重写普通正文。
func TestRewriteWikiLinks_SkipsCodeBlocks(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "`[[Old Title]]` inline\n\n```\n[[Old Title]]\n```\n\n[[Old Title]] outside"

	result := engine.RewriteWikiLinks(content, []RewriteRule{
		{OldTitle: "Old Title", NewTitle: "New Title"},
	})

	want := "`[[Old Title]]` inline\n\n```\n[[Old Title]]\n```\n\n[[New Title]] outside"
	if result.Content != want {
		t.Errorf("got %q, want %q", result.Content, want)
	}
	if result.Count() != 1 {
		t.Errorf("expected 1 replacement (outside code), got %d", result.Count())
	}
}

// 非匹配标题保持正文原值，并且不产生替换计划。
func TestRewriteWikiLinks_NoMatchReturnsUnchanged(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "See [[Other Page]] here."

	result := engine.RewriteWikiLinks(content, []RewriteRule{
		{OldTitle: "Project Plan", NewTitle: "Project Overview"},
	})

	if result.Content != content {
		t.Errorf("content should be unchanged, got %q", result.Content)
	}
	if result.Count() != 0 {
		t.Errorf("expected 0 replacements, got %d", result.Count())
	}
}

// 没有规则时保留完整正文，不因存在双链而主动转换语法。
func TestRewriteWikiLinks_EmptyRulesNoOp(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "[[Some Page]]"
	result := engine.RewriteWikiLinks(content, nil)
	if result.Content != content {
		t.Errorf("expected unchanged content, got %q", result.Content)
	}
}

// 下列用例检查预览标签的返回，不执行正文替换。

// 目标路径对应的提示存在时，预览返回不含开头斜杠的标准双链标签。
func TestFindWikiLinksForPath_FindsPathHint(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "See [[docs/intro]] for details."
	found := engine.FindWikiLinksForPath(content, "/docs/intro", "Intro")
	if len(found) == 0 {
		t.Fatal("expected to find path hint [[docs/intro]]")
	}
	if found[0] != "[[docs/intro]]" {
		t.Errorf("got %q, want [[docs/intro]]", found[0])
	}
}

// 按页面标题命中后返回标题标准标签，不要求正文包含页面路由。
func TestFindWikiLinksForPath_FindsTitleLink(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "See [[Project Plan]] for details."
	found := engine.FindWikiLinksForPath(content, "/docs/project-plan", "Project Plan")
	if len(found) == 0 {
		t.Fatal("expected to find title link [[Project Plan]]")
	}
	if found[0] != "[[Project Plan]]" {
		t.Errorf("got %q, want [[Project Plan]]", found[0])
	}
}

// 正文标题大小写与页面标题不同仍能命中预览查找。
func TestFindWikiLinksForPath_TitleMatchCaseInsensitive(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "See [[project plan]] for details."
	found := engine.FindWikiLinksForPath(content, "/docs/project-plan", "Project Plan")
	if len(found) == 0 {
		t.Fatal("expected case-insensitive title match")
	}
}

// 围栏、行内代码和代码外正文分别调用查找器，确认只有代码外样本命中。
func TestFindWikiLinksForPath_SkipsCodeBlocks(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	// 目标仅出现在围栏内容中，预览不得报告命中。
	fenced := "```\n[[Project Plan]]\n```"
	if found := engine.FindWikiLinksForPath(fenced, "/docs/project-plan", "Project Plan"); len(found) != 0 {
		t.Errorf("fenced block: expected no match, got %v", found)
	}
	// 行内代码使用同一排除机制。
	inline := "`[[Project Plan]]`"
	if found := engine.FindWikiLinksForPath(inline, "/docs/project-plan", "Project Plan"); len(found) != 0 {
		t.Errorf("inline code: expected no match, got %v", found)
	}
	// 相同目标位于普通正文中时应被查到。
	outside := "See [[Project Plan]] here."
	if found := engine.FindWikiLinksForPath(outside, "/docs/project-plan", "Project Plan"); len(found) == 0 {
		t.Errorf("outside code: expected match, got none")
	}
}

// 空正文不产生预览标签，断言只检查长度，没有区分 nil 和空切片。
func TestFindWikiLinksForPath_EmptyContentReturnsNil(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	found := engine.FindWikiLinksForPath("", "/docs/intro", "Intro")
	if len(found) != 0 {
		t.Errorf("expected nil for empty content, got %v", found)
	}
}

// 普通 Markdown 链接不属于此双链预览入口的匹配范围。
func TestFindWikiLinksForPath_NoMatchReturnsEmpty(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "No wiki links here, just [regular](/links)."
	found := engine.FindWikiLinksForPath(content, "/docs/intro", "Intro")
	if len(found) != 0 {
		t.Errorf("expected no matches, got %v", found)
	}
}

// 标题与路径提示相同时，两类规则会覆盖同一原文区间。
// 重叠筛选保留排在前面的标题替换，每个出现位置只计一次，别名仍然保留。
func TestRewriteWikiLinks_TitleEqualsPathHint(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	content := "See [[intro]] and [[intro|Start]] here."

	result := engine.RewriteWikiLinks(content, []RewriteRule{{
		OldPath:  "/intro",
		NewPath:  "/introduction",
		OldTitle: "intro",
		NewTitle: "Introduction",
	}})

	want := "See [[Introduction]] and [[Introduction|Start]] here."
	if result.Content != want {
		t.Errorf("got %q, want %q", result.Content, want)
	}
	if result.Count() != 2 {
		t.Errorf("expected 2 replacements, got %d", result.Count())
	}
}

// 标题未变化时不生成标题替换，但路径变化仍应生成路径提示替换。
func TestRewriteWikiLinks_PathOnlyMoveWithSameTitle(t *testing.T) {
	engine := NewMarkdownRefactorEngine()
	// 两个输入目标都等于旧路径提示，预期改成带 docs 层级的新路径。
	content := "See [[intro]] and [[intro|Alias]] here."

	result := engine.RewriteWikiLinks(content, []RewriteRule{{
		OldPath:  "/intro",
		NewPath:  "/docs/intro",
		OldTitle: "intro",
		NewTitle: "intro",
	}})

	want := "See [[docs/intro]] and [[docs/intro|Alias]] here."
	if result.Content != want {
		t.Errorf("got %q, want %q", result.Content, want)
	}
	if result.Count() != 2 {
		t.Errorf("expected 2 replacements, got %d", result.Count())
	}
}
