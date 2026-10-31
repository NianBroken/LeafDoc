package tree

import (
	"os"
	"sort"
	"strings"
)

// sortReconstructionEntries 原地稳定排序目录项，先比较小写文件名。
// 小写形式相同时再比较原文件名，使大小写不同的名称也获得确定顺序。
// 比较只读取名称，不区分目录和文件，也不查询文件内容。
func sortReconstructionEntries(entries []os.DirEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i].Name(), entries[j].Name()
		la, lb := strings.ToLower(a), strings.ToLower(b)
		if la == lb {
			return a < b
		}
		return la < lb
	})
}
