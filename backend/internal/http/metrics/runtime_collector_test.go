package metrics

import (
	"errors"
	"strings"
	"testing"
)

// fakeRuntimeSource 分别控制页面与用户来源的返回值和错误，不读取真实存储。
type fakeRuntimeSource struct {
	pages    PageStats
	pagesErr error
	users    UserStats
	usersErr error
}

// TestHTTPMetrics_RegisterRuntimeStats_SkipsGaugesWhenSourceErrors 使两个来源同时失败。
// 指标端点仍返回成功并包含构建信息，页面及按角色统计的用户样本均不导出。
func TestHTTPMetrics_RegisterRuntimeStats_SkipsGaugesWhenSourceErrors(t *testing.T) {
	m := NewHTTPMetrics("test")
	m.RegisterRuntimeStats(fakeRuntimeSource{
		pagesErr: errors.New("tree not loaded"),
		usersErr: errors.New("user store unavailable"),
	})

	// 共享读取助手验证 200，这里再确认失败不影响其他指标的输出。
	body := getMetricsBody(t, m.HTTPHandler())
	if !strings.Contains(body, "leafdoc_build_info") {
		t.Fatalf("expected the rest of /metrics to keep working, got:\n%s", body)
	}
	if strings.Contains(body, "leafdoc_pages{") || strings.Contains(body, "leafdoc_users{") {
		t.Errorf("expected no runtime gauge samples when source errors, got:\n%s", body)
	}
}

// TestHTTPMetrics_RegisterRuntimeStats_NilReceiverIsNoop 确认关闭指标时仍可调用注册入口。
func TestHTTPMetrics_RegisterRuntimeStats_NilReceiverIsNoop(t *testing.T) {
	var m *HTTPMetrics
	m.RegisterRuntimeStats(fakeRuntimeSource{}) // 空接收者直接返回，不触发 panic。
}

// PageStats 返回独立配置的页面快照与错误，供采集器验证失败隔离。
func (f fakeRuntimeSource) PageStats() (PageStats, error) { return f.pages, f.pagesErr }

// UserStats 返回独立配置的用户快照与错误，不受页面读取结果影响。
func (f fakeRuntimeSource) UserStats() (UserStats, error) { return f.users, f.usersErr }
