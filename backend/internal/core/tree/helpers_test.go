package tree

import (
	"errors"
	"testing"
	"time"
)

// TestNodeVersionRules 使用同一个节点依次检查版本缺失、不匹配、匹配和显式跳过校验。
// 错误使用 errors.Is 识别业务分类，成功分支必须返回空错误。
func TestNodeVersionRules(t *testing.T) {
	n := &PageNode{Metadata: PageMetadata{UpdatedAt: time.Now().UTC()}}
	if err := checkNodeVersion(n, ""); !errors.Is(err, ErrVersionRequired) {
		t.Fatal(err)
	}
	if err := checkNodeVersion(n, "old"); !errors.Is(err, ErrVersionConflict) {
		t.Fatal(err)
	}
	if err := checkNodeVersion(n, n.Version()); err != nil {
		t.Fatal(err)
	}
	if err := checkNodeVersion(n, VersionUnchecked); err != nil {
		t.Fatal(err)
	}
}

// TestLogicalPaths 在内存中连接三级节点，验证逻辑路径包含虚拟根节点的路径段。
// 此函数生成的是逻辑路径，不访问文件系统或检查页面记录。
func TestLogicalPaths(t *testing.T) {
	root := &PageNode{ID: "root", Slug: "root"}
	a := &PageNode{Slug: "docs", Parent: root}
	b := &PageNode{Slug: "guide", Parent: a}
	if got := GeneratePathFromPageNode(b); got != "root/docs/guide" {
		t.Fatal(got)
	}
}
