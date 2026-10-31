package metrics

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// getMetricsBody 在内存中调用指标处理器，要求返回 200 后交出完整文本供样本断言。
func getMetricsBody(t *testing.T, handler http.Handler) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d: %s", rec.Code, rec.Body.String())
	}

	return rec.Body.String()
}

// TestHTTPMetrics_ExportsPrometheusMetrics 发出一次健康请求，确认导出包含次数、耗时与活动数指标。
func TestHTTPMetrics_ExportsPrometheusMetrics(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	metrics := NewHTTPMetrics("test")
	router := gin.New()
	router.Use(metrics.Middleware())
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /health, got %d: %s", rec.Code, rec.Body.String())
	}

	body := getMetricsBody(t, metrics.HTTPHandler())
	if !strings.Contains(body, "leafdoc_http_requests_total") {
		t.Fatalf("expected metrics output to contain request counter, got: %s", body)
	}
	if !strings.Contains(body, "leafdoc_http_request_duration_seconds") {
		t.Fatalf("expected metrics output to contain duration histogram, got: %s", body)
	}
	if !strings.Contains(body, "leafdoc_http_requests_in_flight") {
		t.Fatalf("expected metrics output to contain in-flight gauge, got: %s", body)
	}
}

// TestHTTPMetrics_ExportsBuildInfoWithVersionLabel 核对版本标签与恒为 1 的构建信息样本。
func TestHTTPMetrics_ExportsBuildInfoWithVersionLabel(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	metrics := NewHTTPMetrics("v0.12.0")

	body := getMetricsBody(t, metrics.HTTPHandler())
	expected := `leafdoc_build_info{version="v0.12.0"} 1`
	if !strings.Contains(body, expected) {
		t.Fatalf("expected metrics output to contain %q, got: %s", expected, body)
	}
}

// TestHTTPMetrics_ExportsGoRuntimeAndProcessMetrics 检查标准运行时样本，并在 Linux 检查进程样本。
func TestHTTPMetrics_ExportsGoRuntimeAndProcessMetrics(t *testing.T) {
	metrics := NewHTTPMetrics("test")

	body := getMetricsBody(t, metrics.HTTPHandler())

	for _, want := range []string{"go_goroutines", "go_gc_duration_seconds", "go_memstats_alloc_bytes"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected Go runtime metric %q in /metrics output, got: %s", want, body)
		}
	}

	// 此测试仅在 Linux 要求进程指标存在，其他平台不对这些样本作断言。
	if runtime.GOOS == "linux" {
		for _, want := range []string{"process_resident_memory_bytes", "process_cpu_seconds_total", "process_open_fds"} {
			if !strings.Contains(body, want) {
				t.Fatalf("expected process metric %q in /metrics output, got: %s", want, body)
			}
		}
	}
}

// TestHTTPMetrics_UsesNormalizedRouteLabel 核对已注册固定路径的方法、路由与状态标签。
func TestHTTPMetrics_UsesNormalizedRouteLabel(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	metrics := NewHTTPMetrics("test")
	router := gin.New()
	router.Use(metrics.Middleware())
	router.GET("/api/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/health, got %d: %s", rec.Code, rec.Body.String())
	}

	body := getMetricsBody(t, metrics.HTTPHandler())
	expected := `leafdoc_http_requests_total{method="GET",route="/api/health",status="200"} 1`
	if !strings.Contains(body, expected) {
		t.Fatalf("expected metrics output to contain %q, got: %s", expected, body)
	}
}

// TestHTTPMetrics_UsesRoutePatternForPathParams 确认标签使用 :id 模式，不泄入具体页面标识。
func TestHTTPMetrics_UsesRoutePatternForPathParams(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	metrics := NewHTTPMetrics("test")
	router := gin.New()
	router.Use(metrics.Middleware())
	router.GET("/api/pages/:id", func(c *gin.Context) {
		c.String(http.StatusOK, c.Param("id"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/pages/abc123", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/pages/:id, got %d: %s", rec.Code, rec.Body.String())
	}

	body := getMetricsBody(t, metrics.HTTPHandler())
	expected := `leafdoc_http_requests_total{method="GET",route="/api/pages/:id",status="200"} 1`
	if !strings.Contains(body, expected) {
		t.Fatalf("expected metrics output to contain %q, got: %s", expected, body)
	}
	if strings.Contains(body, "/api/pages/abc123") {
		t.Fatalf("expected metrics output not to contain concrete page ID, got: %s", body)
	}
}

// TestHTTPMetrics_UsesUnmatchedLabelForUnknownRoutes 确认未知路径汇总为 unmatched 且保留 404 状态。
func TestHTTPMetrics_UsesUnmatchedLabelForUnknownRoutes(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	metrics := NewHTTPMetrics("test")
	router := gin.New()
	router.Use(metrics.Middleware())

	req := httptest.NewRequest(http.MethodGet, "/api/does-not-exist/123", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown route, got %d: %s", rec.Code, rec.Body.String())
	}

	body := getMetricsBody(t, metrics.HTTPHandler())
	expected := `leafdoc_http_requests_total{method="GET",route="unmatched",status="404"} 1`
	if !strings.Contains(body, expected) {
		t.Fatalf("expected metrics output to contain %q, got: %s", expected, body)
	}
}
