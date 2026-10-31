package wiki

import (
	"github.com/NianBroken/LeafDoc/internal/core/auth"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// runtimeStatsSource 将 Wiki 的树与用户服务适配为指标采集时使用的数据来源。
// 保存 Wiki 而非固定用户服务，使恢复后采集仍通过 UserService 取得当前存储。
type runtimeStatsSource struct {
	w *Wiki
}

// PageStats 直接取得树内存中的节点数量，不在这里扫描页面正文或执行数据库查询。
func (s runtimeStatsSource) PageStats() (httpmetrics.PageStats, error) {
	pages, sections := s.w.tree.NodeCounts()
	return httpmetrics.PageStats{Pages: pages, Sections: sections}, nil
}

// UserStats 每次取得当前用户列表，按实际角色累计账号，并独立统计必须设置密码的账号。
// 列表读取失败返回错误，不导出不完整的用户统计。
func (s runtimeStatsSource) UserStats() (httpmetrics.UserStats, error) {
	users, err := s.w.UserService().GetUsers()
	if err != nil {
		return httpmetrics.UserStats{}, err
	}

	// 三个已知角色预置零值，列表中出现其他角色时映射也会为其形成独立计数。
	stats := httpmetrics.UserStats{
		ByRole: map[string]int{
			auth.RoleAdmin:  0,
			auth.RoleEditor: 0,
			auth.RoleViewer: 0,
		},
	}
	for _, u := range users {
		stats.ByRole[u.Role]++
		if u.MustSetPassword {
			stats.PendingInvite++
		}
	}
	return stats, nil
}
