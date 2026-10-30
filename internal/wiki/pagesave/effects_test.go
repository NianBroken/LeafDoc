package pagesave

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

func TestNewTagsSideEffect_DefaultsLogger(t *testing.T) {
	effect := NewTagsSideEffect(nil, nil, nil)
	if effect.log == nil {
		t.Fatal("expected default logger to be set")
	}
	if effect.log != slog.Default() {
		t.Fatal("expected slog.Default() logger")
	}
}

func TestNewPropertiesSideEffect_DefaultsLogger(t *testing.T) {
	effect := NewPropertiesSideEffect(nil, nil, nil)
	if effect.log == nil {
		t.Fatal("expected default logger to be set")
	}
	if effect.log != slog.Default() {
		t.Fatal("expected slog.Default() logger")
	}
}

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

type metricTestEffect struct {
	metrics *httpmetrics.HTTPMetrics
}

func (e metricTestEffect) Name() string {
	return "search"
}

func (e metricTestEffect) Apply(event PageSaveEvent) {
	e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
}
