package metrics

import (
	"log/slog"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
)

// PageStats 保存一次采集取得的页面与章节数量，不在结构体内部持续跟踪变化。
type PageStats struct {
	Pages    int
	Sections int
}

// UserStats 保存按角色分组的账号数与待完成邀请数。
// 采集器只导出 ByRole 已有键，需要保留零数量角色的调用方必须显式提供对应键。
type UserStats struct {
	ByRole        map[string]int
	PendingInvite int
}

// RuntimeStatsSource 提供每次指标采集需要的当前内容及账号数量。
// 两个方法在采集时调用，任一返回错误仅跳过该组样本，不使另一组失败。
type RuntimeStatsSource interface {
	PageStats() (PageStats, error)
	UserStats() (UserStats, error)
}

// runtimeCollector 在每次采集时读取来源数据，并将当前数量作为 Gauge 样本发送。
// 它不缓存上次结果，也不把当前总量错误地累计为只能增加的 Counter。
type runtimeCollector struct {
	src RuntimeStatsSource
	log *slog.Logger

	pages        *prometheus.Desc
	users        *prometheus.Desc
	usersPending *prometheus.Desc
}

// newRuntimeCollector 保存数据来源并建立三个指标描述，不在构造时读取业务数据。
func newRuntimeCollector(src RuntimeStatsSource) *runtimeCollector {
	return &runtimeCollector{
		src: src,
		log: slog.Default().With("component", "metrics.runtime"),
		pages: prometheus.NewDesc(
			"leafdoc_pages",
			"Current number of nodes in the page tree by kind (page or section).",
			[]string{"kind"}, nil,
		),
		users: prometheus.NewDesc(
			"leafdoc_users",
			"Current number of user accounts by role.",
			[]string{"role"}, nil,
		),
		usersPending: prometheus.NewDesc(
			"leafdoc_users_pending_invite",
			"Current number of invited user accounts that have not yet completed sign-up.",
			nil, nil,
		),
	}
}

// Describe 向注册表声明全部指标，即使某次来源错误导致没有对应样本也保持描述稳定。
func (c *runtimeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.pages
	ch <- c.users
	ch <- c.usersPending
}

// Collect 顺序读取页面与用户统计，各来源错误按 Debug 记录且不导出该组样本。
// 页面读取失败后仍继续读取用户；用户失败时直接结束，此前已发送的页面样本保留。
func (c *runtimeCollector) Collect(ch chan<- prometheus.Metric) {
	if ps, err := c.src.PageStats(); err != nil {
		c.log.Debug("skipping page gauges for this scrape", "error", err)
	} else {
		ch <- prometheus.MustNewConstMetric(c.pages, prometheus.GaugeValue, float64(ps.Pages), "page")
		ch <- prometheus.MustNewConstMetric(c.pages, prometheus.GaugeValue, float64(ps.Sections), "section")
	}

	us, err := c.src.UserStats()
	if err != nil {
		c.log.Debug("skipping user gauges for this scrape", "error", err)
		return
	}

	// 映射先收集键并排序，使同一组角色的发送顺序稳定。
	roles := make([]string, 0, len(us.ByRole))
	for role := range us.ByRole {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		ch <- prometheus.MustNewConstMetric(c.users, prometheus.GaugeValue, float64(us.ByRole[role]), role)
	}

	ch <- prometheus.MustNewConstMetric(c.usersPending, prometheus.GaugeValue, float64(us.PendingInvite))
}

// RegisterRuntimeStats 将业务数量采集器注册到实例的独立注册表。
// 空实例或空来源直接返回，正常实例应只注册一次，重复描述交给 MustRegister 报错。
func (m *HTTPMetrics) RegisterRuntimeStats(src RuntimeStatsSource) {
	if m == nil || src == nil {
		return
	}
	m.registry.MustRegister(newRuntimeCollector(src))
}
