package pages

import (
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// sanitizeClientVersion 将客户端提供的内部跳过版本标记转为空字符串。
// 有版本的节点随后按缺少版本处理，外部请求不能用该标记绕过并发校验。
func sanitizeClientVersion(v string) string {
	if v == tree.VersionUnchecked {
		return ""
	}
	return v
}

// collectSubtreeIDs 按深度优先先序收集标识，跳过 root 但继续遍历其子节点。
// 空节点不进入结果，此处不检测循环或重复标识。
func collectSubtreeIDs(node *tree.PageNode) []string {
	var ids []string
	var walk func(n *tree.PageNode)
	walk = func(n *tree.PageNode) {
		if n == nil {
			return
		}
		if n.ID != "root" {
			ids = append(ids, n.ID)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(node)
	return ids
}

// deleteRevisionData 按输入顺序删除页面历史，首个失败立即返回，不处理后续标识。
// 未配置历史服务时直接成功，不对标识列表执行去重。
func deleteRevisionData(svc *revision.Service, pageIDs []string) error {
	if svc == nil {
		return nil
	}
	for _, id := range pageIDs {
		if err := svc.DeletePageData(id); err != nil {
			return err
		}
	}
	return nil
}
