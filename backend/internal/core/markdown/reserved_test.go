// 本文件用表格案例核验系统键分类，不修改或序列化实际页面字段。
package markdown

import "testing"

// TestIsSystemKey 覆盖标签大小写、系统前缀、普通属性和空键。
// 每个输入独立运行子测试，错误输出保留具体键名以定位分类差异。
func TestIsSystemKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"tags", true},
		{"Tags", true},
		{"TAGS", true},
		{"leafdoc_id", true},
		{"leafdoc_title", true},
		{"leafdoc_created_at", true},
		{"LEAFDOC_ANYTHING", true},
		// title 属于普通属性，可与 leafdoc_title 同时存在，不被系统前缀规则排除。
		{"title", false},
		{"Title", false},
		{"status", false},
		{"author", false},
		{"permalink", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			got := IsSystemKey(tc.key)
			if got != tc.want {
				t.Fatalf("IsSystemKey(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}
