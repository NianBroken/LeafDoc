// 本文件集中判断页首元信息中的系统保留键，供属性读取和编辑相关代码使用。
package markdown

import "strings"

// IsSystemKey 去掉两端空白并转为小写后，识别 tags 和 leafdoc_ 前缀。
// title 不满足这两个条件，因此本函数将它视为普通键。
// 这里只返回分类结果，不删除字段、不改变原键，也不直接限制编辑权限。
func IsSystemKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	return lower == "tags" || strings.HasPrefix(lower, "leafdoc_")
}
