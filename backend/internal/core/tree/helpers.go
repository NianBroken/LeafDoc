// 本文件集中执行节点版本校验并提供逐级 slug 路径拼接辅助函数。
package tree

// checkNodeVersion 供持写锁的操作比较客户端版本，本函数自身不获取锁。
// 显式跳过值或节点空版本直接通过，节点有版本但请求为空返回缺少版本错误。
// 两个非空版本不相同时返回冲突，不在此修改更新时间或重试写入。
func checkNodeVersion(node *PageNode, expectedVersion string) error {
	if expectedVersion == VersionUnchecked {
		return nil
	}
	nodeVersion := node.Version()
	if nodeVersion == "" {
		return nil
	}
	if expectedVersion == "" {
		return ErrVersionRequired
	}
	if nodeVersion != expectedVersion {
		return ErrVersionConflict
	}
	return nil
}

// GeneratePathFromPageNode 递归拼接祖先 slug，根节点 slug 原样保留。
// 与 CalculatePath 不同，此函数不把 root 或空根 slug 做特殊省略处理。
func GeneratePathFromPageNode(entry *PageNode) string {
	path := ""
	if entry.Parent != nil {
		path = GeneratePathFromPageNode(entry.Parent) + "/" + entry.Slug
	} else {
		path = entry.Slug
	}
	return path
}
