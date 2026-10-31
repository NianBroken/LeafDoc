// 本文件测量双链预览在单页及百页批次中的匹配式编译成本。
// 样本同时含标题、路径提示和代码内双链，结果用于性能测量，不断言功能输出。
package links

import (
	"strings"
	"testing"
)

// 单页循环复用循环外编译的匹配式，主要测量正文解析和查找本身。
func BenchmarkFindWikiLinksPreview(b *testing.B) {
	e := NewMarkdownRefactorEngine()
	finder := CompileWikiLinkFinder("/docs/project-plan", "Project Plan")
	content := strings.Repeat("Text [[Project Plan|Alias]] and [[docs/project-plan]]. `[[Project Plan]]`\n\n", 20)
	b.ReportAllocs()
	for b.Loop() {
		e.FindWikiLinksPrecompiled(content, finder)
	}
}

// 每轮模拟一百页预览，对比每页编译与每批编译一次的路径。
// 两条分支共用同样的正文和解析引擎，均报告内存分配。
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
				// 批次分支在每轮基准内编译一次，再供百次页面查找复用。
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
