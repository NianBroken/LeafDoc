package links

import (
	"strings"
	"testing"
)

func BenchmarkFindWikiLinksPreview(b *testing.B) {
	e := NewMarkdownRefactorEngine()
	finder := CompileWikiLinkFinder("/docs/project-plan", "Project Plan")
	content := strings.Repeat("Text [[Project Plan|Alias]] and [[docs/project-plan]]. `[[Project Plan]]`\n\n", 20)
	b.ReportAllocs()
	for b.Loop() {
		e.FindWikiLinksPrecompiled(content, finder)
	}
}

func BenchmarkFindWikiLinksPreviewBatch(b *testing.B) {
	content := strings.Repeat("Text [[Project Plan|Alias]] and [[docs/project-plan]]. `[[Project Plan]]`\n\n", 20)
	for _, compiled := range []bool{false, true} {
		name := "CompilePerPage"
		if compiled {
			name = "CompilePerPreview"
		}
		b.Run(name, func(b *testing.B) {
			e := NewMarkdownRefactorEngine()
			b.ReportAllocs()
			for b.Loop() {
				if compiled {
					finder := CompileWikiLinkFinder("/docs/project-plan", "Project Plan")
					for i := 0; i < 100; i++ {
						e.FindWikiLinksPrecompiled(content, finder)
					}
				} else {
					for i := 0; i < 100; i++ {
						e.FindWikiLinksForPath(content, "/docs/project-plan", "Project Plan")
					}
				}
			}
		})
	}
}
