// 本文件定义路径确保操作返回的目标节点及创建记录。
package tree

// EnsurePathResult 组合最终节点、存在状态以及处理过程中创建的节点列表。
// Created 保存节点引用，不是独立的页面内容副本。
type EnsurePathResult struct {
	Page    *PageNode
	Exists  bool
	Created []*PageNode
}
