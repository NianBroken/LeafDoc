package pagesave

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// TestNewTagsSideEffect_DefaultsLogger 验证空日志参数使用系统默认日志对象。
func TestNewTagsSideEffect_DefaultsLogger(t *testing.T) {
	effect := NewTagsSideEffect(nil, nil, nil)
	if effect.log == nil {
		t.Fatal("expected default logger to be set")
	}
	if effect.log != slog.Default() {
		t.Fatal("expected slog.Default() logger")
	}
}

// TestNewPropertiesSideEffect_DefaultsLogger 独立检查属性处理器的日志回退行为。
func TestNewPropertiesSideEffect_DefaultsLogger(t *testing.T) {
	effect := NewPropertiesSideEffect(nil, nil, nil)
	if effect.log == nil {
		t.Fatal("expected default logger to be set")
	}
	if effect.log != slog.Default() {
		t.Fatal("expected slog.Default() logger")
	}
}

// TestPageSaveOrchestrator_RecordsSideEffectDurationAndFailures
// 通过普通 Apply 替身检查耗时和失败计数可从实际指标接口读取。
// 失败计数由替身主动递增，此处不模拟事务处理器返回错误。
func TestPageSaveOrchestrator_RecordsSideEffectDurationAndFailures(t *testing.T) {
	metrics := httpmetrics.NewHTTPMetrics("test")
	orchestrator := NewPageSaveOrchestrator(metrics, metricTestEffect{metrics: metrics})

	if err := orchestrator.Run(PageSaveEvent{Operation: PageOperationUpdate}); err != nil {
		t.Fatal(err)
	}

	body := metricsBody(t, metrics)
	if !strings.Contains(body, `leafdoc_pagesave_sideeffect_duration_seconds_bucket{operation="update",side_effect="search"`) {
		t.Fatalf("expected side-effect duration metric, got: %s", body)
	}
	expectedFailure := `leafdoc_pagesave_sideeffect_failures_total{operation="update",side_effect="search"} 1`
	if !strings.Contains(body, expectedFailure) {
		t.Fatalf("expected side-effect failure metric %q, got: %s", expectedFailure, body)
	}
}

// metricsBody 通过内存 HTTP 请求调用实际指标处理器，响应成功后返回正文。
func metricsBody(t *testing.T, metrics *httpmetrics.HTTPMetrics) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	metrics.HTTPHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics endpoint to return 200, got %d", rec.Code)
	}
	return rec.Body.String()
}

// metricTestEffect 使用 search 名称验证指标，不操作真实搜索索引。
type metricTestEffect struct {
	metrics *httpmetrics.HTTPMetrics
}

// Name 为指标提供固定处理器维度。
func (e metricTestEffect) Name() string {
	return "search"
}

// Apply 每次递增一次失败计数，返回后编排器继续记录完成耗时。
func (e metricTestEffect) Apply(event PageSaveEvent) {
	e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
}
