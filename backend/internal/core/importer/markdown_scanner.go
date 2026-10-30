package importer

import (
	"strings"
)

func findFenceEnd(content string, start int, fenceChar byte, fenceLen int) int {
	i := start
	for {
		lineEnd := i
		for lineEnd < len(content) && content[lineEnd] != '\n' {
			lineEnd++
		}

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
		searchFrom = next + 1
	}
	return len(content)
}

func isLineStart(content string, index int) bool {
	return index == 0 || content[index-1] == '\n'
}

func countRepeatedByte(content string, start int, target byte) int {
	count := 0
	for start+count < len(content) && content[start+count] == target {
		count++
	}
	return count
}

func countLeadingByte(content string, target byte) int {
	count := 0
	for count < len(content) && content[count] == target {
		count++
	}
	return count
}
