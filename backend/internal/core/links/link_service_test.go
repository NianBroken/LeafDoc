// 本文件使用正文样本验证普通链接与双链的提取规则。测试不连接索引数据库，
// 只检查过滤、目标文本、顺序和去重，不验证目标页面是否实际存在。
package links

import (
	"testing"
)

// 混合相对路径、绝对路径、查询串和片段，确认保留站内目标的原顺序，
// 去掉查询串与片段，并排除纯锚点、网页地址和邮件地址。
func TestExtractLinksFromMarkdown_FiltersExternalAndNormalizes(t *testing.T) {
	md := `
# Example

Internal: [Page 1](/docs/page1)
Relative: [Rel](../docs/page2)
Anchor only: [Section](#heading)
External: [Google](https://google.com)
Mail: [Mail](mailto:test@example.com)
With fragment: [WithFragment](/docs/page3#intro)
With query: [WithQuery](/docs/page4?foo=bar)
With both: [Both](/docs/page5?foo=bar#section)
`

	links := extractLinksFromMarkdown(md)

	want := []string{
		"/docs/page1",
		"../docs/page2",
		"/docs/page3",
		"/docs/page4",
		"/docs/page5",
	}

	if len(links) != len(want) {
		t.Fatalf("expected %d links, got %d: %#v", len(want), len(links), links)
	}

	for i, w := range want {
		if links[i] != w {
			t.Errorf("link[%d] = %q, want %q", i, links[i], w)
		}
	}
}

// 大写网页协议和混合大小写邮件协议仍应被过滤，普通站内目标保留。
func TestExtractLinksFromMarkdown_IgnoresExternalLinksCaseInsensitive(t *testing.T) {
	md := `
[HTTPS](HTTPS://example.com)
[Mail](Mailto:test@example.com)
[Anchor](#intro)
[Wiki](/docs/page1)
`

	links := extractLinksFromMarkdown(md)

	if len(links) != 1 {
		t.Fatalf("expected 1 wiki link, got %d: %#v", len(links), links)
	}
	if links[0] != "/docs/page1" {
		t.Fatalf("link[0] = %q, want %q", links[0], "/docs/page1")
	}
}

// 两种受支持的附件目录前缀均不进入页面出链，其他页面目标仍可提取。
func TestExtractLinksFromMarkdown_IgnoresAssetDestinations(t *testing.T) {
	md := `
Asset absolute: [File](/assets/abc/manual.pdf)
Asset relative: [Image](assets/abc/picture.png)
Internal: [Page](/docs/page1)
`

	links := extractLinksFromMarkdown(md)

	want := []string{"/docs/page1"}
	if len(links) != len(want) {
		t.Fatalf("expected %d links, got %d: %#v", len(want), len(links), links)
	}
	for i, w := range want {
		if links[i] != w {
			t.Fatalf("link[%d] = %q, want %q", i, links[i], w)
		}
	}
}

// 进一步覆盖 HTTP、HTTPS 和 MAILTO 的不同大小写组合，避免过滤只接受小写文本。
func TestExtractLinksFromMarkdown_IgnoresMixedCaseExternalSchemes(t *testing.T) {
	md := `
Uppercase HTTPS: [A](HTTPS://example.com)
Uppercase HTTP: [B](HTTP://example.com)
Uppercase MAILTO: [C](MAILTO:foo@bar.com)
Mixed case Https: [D](Https://example.com)
Mixed case Mailto: [E](Mailto:foo@bar.com)
Internal: [Page](/docs/page1)
`

	links := extractLinksFromMarkdown(md)

	want := []string{"/docs/page1"}
	if len(links) != len(want) {
		t.Fatalf("expected %d links, got %d: %#v", len(want), len(links), links)
	}
	for i, w := range want {
		if links[i] != w {
			t.Fatalf("link[%d] = %q, want %q", i, links[i], w)
		}
	}
}

// 双链返回目标而不是别名，含斜杠目标保持原文，普通 Markdown 链接不混入结果。
func TestExtractWikiLinksFromMarkdown_BasicSyntax(t *testing.T) {
	md := `
See [[Project Plan]] for details.
Also check [[Meeting Notes|our last meeting]].
And [[Folder/SubPage]] path hint.
Normal [link](/docs/page) stays untouched.
`
	got := extractWikiLinksFromMarkdown(md)
	want := []string{"Project Plan", "Meeting Notes", "Folder/SubPage"}
	if len(got) != len(want) {
		t.Fatalf("expected %d wiki links, got %d: %v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// 同一目标出现多次或使用不同别名时，只保留一项目标文本。
func TestExtractWikiLinksFromMarkdown_DeduplicatesTargets(t *testing.T) {
	md := `[[Notes]] and [[Notes]] again and [[Notes|different alias]].`
	got := extractWikiLinksFromMarkdown(md)
	if len(got) != 1 {
		t.Fatalf("expected 1 deduplicated entry, got %d: %v", len(got), got)
	}
	if got[0] != "Notes" {
		t.Errorf("got %q, want %q", got[0], "Notes")
	}
}

// 围栏代码中的 shell 条件样本不得形成页面双链，结果必须为空。
func TestExtractWikiLinksFromMarkdown_IgnoresFencedCodeBlock(t *testing.T) {
	md := "```bash\nif [[ -n \"$computed_hash\" && -n \"$src_hash\" ]]; then\n  [[ \"$src_hash\" == sha256-* ]]\nfi\n```\n"

	got := extractWikiLinksFromMarkdown(md)
	if len(got) != 0 {
		t.Fatalf("expected no wiki links from fenced code block, got %v", got)
	}
}

// 行内代码包裹的条件表达式不作为双链目标。
func TestExtractWikiLinksFromMarkdown_IgnoresInlineCode(t *testing.T) {
	got := extractWikiLinksFromMarkdown("`[[ -n $x ]]`")
	if len(got) != 0 {
		t.Fatalf("expected no wiki links from inline code, got %v", got)
	}
}

// 同一正文含真实双链与代码样本时，只提取代码区间之外的真实目标。
func TestExtractWikiLinksFromMarkdown_MixedRealAndCodeBlock(t *testing.T) {
	md := "See [[Real Page]].\n\n```\n[[ -z \"$x\" ]]\n```\n"

	got := extractWikiLinksFromMarkdown(md)
	if len(got) != 1 {
		t.Fatalf("expected 1 wiki link, got %d: %v", len(got), got)
	}
	if got[0] != "Real Page" {
		t.Errorf("got %q, want %q", got[0], "Real Page")
	}
}

// 缩进代码样本的双括号内容不形成目标；样本还含目标开头空格，
// 因此本断言不单独证明排除结果只由代码区间识别造成。
func TestExtractWikiLinksFromMarkdown_IgnoresIndentedCodeBlock(t *testing.T) {
	got := extractWikiLinksFromMarkdown("    [[ Something ]]\n")
	if len(got) != 0 {
		t.Fatalf("expected no wiki links from indented code block, got %v", got)
	}
}

// 即使不在代码区间，双括号后的空白也让所列 shell 条件不符合双链正则。
func TestExtractWikiLinksFromMarkdown_IgnoresBashConditionalsOutsideCodeBlocks(t *testing.T) {
	cases := []string{
		`if [[ -n "$computed_hash" && -n "$src_hash" ]]; then`,
		`if [[ "$src_hash" == sha256-* ]]; then`,
	}
	for _, content := range cases {
		got := extractWikiLinksFromMarkdown(content)
		if len(got) != 0 {
			t.Errorf("input %q: expected no wiki links, got %v", content, got)
		}
	}
}

// 完全不含双链结构的正文返回长度为零的结果，不要求 nil 与空切片的具体形式。
func TestExtractWikiLinksFromMarkdown_Empty(t *testing.T) {
	got := extractWikiLinksFromMarkdown("No wiki links here.")
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}
