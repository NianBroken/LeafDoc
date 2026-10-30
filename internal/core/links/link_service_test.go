package links

import (
	"testing"
)

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

func TestExtractWikiLinksFromMarkdown_IgnoresFencedCodeBlock(t *testing.T) {
	md := "```bash\nif [[ -n \"$computed_hash\" && -n \"$src_hash\" ]]; then\n  [[ \"$src_hash\" == sha256-* ]]\nfi\n```\n"

	got := extractWikiLinksFromMarkdown(md)
	if len(got) != 0 {
		t.Fatalf("expected no wiki links from fenced code block, got %v", got)
	}
}

func TestExtractWikiLinksFromMarkdown_IgnoresInlineCode(t *testing.T) {
	got := extractWikiLinksFromMarkdown("`[[ -n $x ]]`")
	if len(got) != 0 {
		t.Fatalf("expected no wiki links from inline code, got %v", got)
	}
}

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

func TestExtractWikiLinksFromMarkdown_IgnoresIndentedCodeBlock(t *testing.T) {
	got := extractWikiLinksFromMarkdown("    [[ Something ]]\n")
	if len(got) != 0 {
		t.Fatalf("expected no wiki links from indented code block, got %v", got)
	}
}

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

func TestExtractWikiLinksFromMarkdown_Empty(t *testing.T) {
	got := extractWikiLinksFromMarkdown("No wiki links here.")
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}
