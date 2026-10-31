package importer

import (
	"strings"
)

// findFenceEnd 从围栏起始行向后查找结束行，返回该行换行符之后的位置。
// 结束行允许最多三个前导空格，连续围栏字符数至少达到起始围栏长度。
// 此处不检查围栏字符后面的文字，未找到结束行时将剩余全文视为代码区域。
func findFenceEnd(content string, start int, fenceChar byte, fenceLen int) int {
	i := start
	for {
		lineEnd := i
		for lineEnd < len(content) && content[lineEnd] != '\n' {
			lineEnd++
		}

		// 起始行只用于确定扫描边界，不参与结束围栏判定。
		if i > start {
			line := content[i:lineEnd]
			trimmed := strings.TrimLeft(line, " ")
			indent := len(line) - len(trimmed)
			if indent <= 3 && len(trimmed) >= fenceLen {
				if countLeadingByte(trimmed, fenceChar) >= fenceLen {
					if lineEnd < len(content) {
						return lineEnd + 1
					}
					return lineEnd
				}
			}
		}

		if lineEnd >= len(content) {
			return len(content)
		}
		i = lineEnd + 1
	}
}

// findInlineCodeEnd 在起始反引号之后查找连续数量与 delimiterLen 相同的位置。
// 返回值包含结束分隔符；找不到匹配位置时返回全文末尾，使剩余文字保持原样。
func findInlineCodeEnd(content string, start int, delimiterLen int) int {
	searchFrom := start + delimiterLen
	for searchFrom < len(content) {
		next := strings.IndexByte(content[searchFrom:], '`')
		if next < 0 {
			return len(content)
		}
		next += searchFrom
		if countRepeatedByte(content, next, '`') == delimiterLen {
			return next + delimiterLen
		}
		// 不匹配时仅前进一个字节，较长反引号串的后续位置仍参与搜索。
		searchFrom = next + 1
	}
	return len(content)
}

// isLineStart 判断当前位置是否位于全文开头或紧随换行符。
// 调用方负责提供有效的字节位置，函数不对越界索引进行检查。
func isLineStart(content string, index int) bool {
	return index == 0 || content[index-1] == '\n'
}

// countRepeatedByte 从指定字节位置累计连续的目标字符，遇到不同字符或末尾停止。
func countRepeatedByte(content string, start int, target byte) int {
	count := 0
	for start+count < len(content) && content[start+count] == target {
		count++
	}
	return count
}

// countLeadingByte 统计字符串开头连续的目标字节，用于判定围栏长度。
func countLeadingByte(content string, target byte) int {
	count := 0
	for count < len(content) && content[count] == target {
		count++
	}
	return count
}
