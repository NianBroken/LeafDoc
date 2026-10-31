// 本文件比较三类正文的双链提取耗时与内存分配，内容均重复两百次形成固定负载。
// 基准结果不作为功能正确性的断言，正文中的注释文本属于输入数据。
package links

import (
	"strings"
	"testing"
)

// 无双链、正文含双链、仅代码含双链分别测试正则快速返回和代码区间排除的成本。
func BenchmarkExtractWikiLinksFastPath(b *testing.B) {
	for _, tc := range []struct{ name, content string }{
		{"NoLinks", strings.Repeat("Paragraph with **formatting**, `code` and [ordinary links](/page).\n\n", 200)},
		{"WithLinks", strings.Repeat("Paragraph [[Overview]] and `[[Excluded]]`.\n\n", 200)},
		{"CodeOnly", strings.Repeat("```go\n// [[Excluded]]\n```\n\n", 200)},
	} {
		// 每种正文独立计时并报告分配，循环只调用提取入口，不重建样本字符串。
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				extractWikiLinksFromMarkdown(tc.content)
			}
		})
	}
}
