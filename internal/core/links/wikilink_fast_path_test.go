package links

import (
	"strings"
	"testing"
)

func BenchmarkExtractWikiLinksFastPath(b *testing.B) {
	for _, tc := range []struct{ name, content string }{
		{"NoLinks", strings.Repeat("Paragraph with **formatting**, `code` and [ordinary links](/page).\n\n", 200)},
		{"WithLinks", strings.Repeat("Paragraph [[Overview]] and `[[Excluded]]`.\n\n", 200)},
		{"CodeOnly", strings.Repeat("```go\n// [[Excluded]]\n```\n\n", 200)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				extractWikiLinksFromMarkdown(tc.content)
			}
		})
	}
}
