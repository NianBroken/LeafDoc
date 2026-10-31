// 本文件以真实树服务验证标题规范化为空时的路径建议，不创建页面或数据库记录。
package pages

import (
	"context"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// TestSuggestSlugUseCase_EmptyNormalizedTitle 验证真实用例对无有效名称的标题返回空建议。
// 空标题不需要遍历父节点冲突列表，因此未加载树的测试服务也能完成这一分支。
func TestSuggestSlugUseCase_EmptyNormalizedTitle(t *testing.T) {
	uc := NewSuggestSlugUseCase(tree.NewTreeService(t.TempDir()), tree.NewSlugService())
	for _, title := range []string{"", " \t", "!!!", "___"} {
		t.Run(title, func(t *testing.T) {
			result, err := uc.Execute(context.Background(), SuggestSlugInput{Title: title})
			if err != nil {
				t.Fatalf("Execute(%q): %v", title, err)
			}
			if result == nil || result.Slug != "" {
				t.Fatalf("Execute(%q) = %#v, want empty slug", title, result)
			}
		})
	}
}
