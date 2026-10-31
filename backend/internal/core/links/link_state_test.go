// 本文件验证标题匹配数量与链接状态的映射，不查询页面树。
package links

import (
	"testing"
)

// 零个匹配为失效，一个为已解析，两个和五个均为歧义，覆盖三个状态分支。
func TestClassifyTitleMatches(t *testing.T) {
	cases := []struct {
		matches int
		want    LinkState
	}{
		{0, LinkBroken},
		{1, LinkResolved},
		{2, LinkAmbiguous},
		{5, LinkAmbiguous},
	}
	for _, tc := range cases {
		if got := classifyTitleMatches(tc.matches); got != tc.want {
			t.Errorf("classifyTitleMatches(%d) = %v, want %v", tc.matches, got, tc.want)
		}
	}
}
