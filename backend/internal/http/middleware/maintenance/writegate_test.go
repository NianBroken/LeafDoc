package maintenance

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/restore"
	"github.com/gin-gonic/gin"
)

// newTestRouter 将所有路径与方法交给同一成功处理函数，令返回码只反映门禁判断。
func newTestRouter(gate *restore.WriteGate) *gin.Engine {
	r := gin.New()
	r.Use(WriteGateMiddleware(gate))
	r.Any("/*path", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

// TestWriteGateMiddleware_AllowsWritesWhenDisengaged 确认开放门禁允许普通页面写请求。
func TestWriteGateMiddleware_AllowsWritesWhenDisengaged(t *testing.T) {
	router := newTestRouter(restore.NewWriteGate())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/pages", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestWriteGateMiddleware_BlocksWritesWhenEngaged 确认关闭门禁时普通 POST 返回 503。
func TestWriteGateMiddleware_BlocksWritesWhenEngaged(t *testing.T) {
	gate := restore.NewWriteGate()
	gate.Engage()
	router := newTestRouter(gate)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/pages", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// TestWriteGateMiddleware_ExemptsGetHeadOptionsEvenWhenEngaged 逐一验证三个免登记方法。
func TestWriteGateMiddleware_ExemptsGetHeadOptionsEvenWhenEngaged(t *testing.T) {
	gate := restore.NewWriteGate()
	gate.Engage()
	router := newTestRouter(gate)

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(method, "/api/pages", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200 even while engaged, got %d", method, w.Code)
		}
	}
}

// TestWriteGateMiddleware_ExemptsRestoreAdminPathEvenWhenEngaged 检查恢复任务与重启子路径仍可调用。
func TestWriteGateMiddleware_ExemptsRestoreAdminPathEvenWhenEngaged(t *testing.T) {
	gate := restore.NewWriteGate()
	gate.Engage()
	router := newTestRouter(gate)

	for _, path := range []string{"/api/admin/restore/snapshot-1", "/api/admin/restore/self-restart"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, path, nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("%s: expected the restore admin path to be exempt from the gate, got %d", path, w.Code)
		}
	}
}

// TestWriteGateMiddleware_StillBlocksOtherAdminPathsWhenEngaged 确认管理员身份路径不整体获得豁免。
func TestWriteGateMiddleware_StillBlocksOtherAdminPathsWhenEngaged(t *testing.T) {
	gate := restore.NewWriteGate()
	gate.Engage()
	router := newTestRouter(gate)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/admin/snapshot", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected non-restore admin writes to still be gated, got %d", w.Code)
	}
}

// TestWriteGateMiddleware_DoesNotMatchUnrelatedRouteWithSimilarName 确认路径段按完整文字比较。
// restore-policy 与 restore 是两个不同段，前者不在恢复路径豁免范围内。
func TestWriteGateMiddleware_DoesNotMatchUnrelatedRouteWithSimilarName(t *testing.T) {
	gate := restore.NewWriteGate()
	gate.Engage()
	router := newTestRouter(gate)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/admin/restore-policy", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected a route that merely contains \"restore\" as a substring (not the exact segment) to still be gated, got %d", w.Code)
	}
}

// TestWriteGateMiddleware_ExemptionSurvivesBasePathPrefix 比较带相同部署前缀的两个路径。
// 恢复接口允许 POST，普通页面接口仍返回 503，前缀不改变路径段匹配规则。
func TestWriteGateMiddleware_ExemptionSurvivesBasePathPrefix(t *testing.T) {
	gate := restore.NewWriteGate()
	gate.Engage()
	router := newTestRouter(gate)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/wiki/api/admin/restore/snapshot-1", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected the restore path to stay exempt behind a --base-path prefix, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/wiki/api/pages", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected an unrelated route behind a --base-path prefix to still be gated, got %d", w.Code)
	}
}

// TestWriteGateMiddleware_RespondsWithLocalizedErrorShape 检查错误体包含代码、消息与翻译模板字段。
// 断言通过文本包含检查完成，不将其描述为完整 JSON 结构校验。
func TestWriteGateMiddleware_RespondsWithLocalizedErrorShape(t *testing.T) {
	gate := restore.NewWriteGate()
	gate.Engage()
	router := newTestRouter(gate)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/pages", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{`"code":"restore_writes_disabled"`, `"message":`, `"template":`} {
		if !strings.Contains(body, want) {
			t.Errorf("expected response body to contain %s (LocalizedError shape), got %s", want, body)
		}
	}
}

// TestRestoreProgressDuringMaintenance 确认进度路径在维护时可读，但不获得写请求豁免。
// 同时覆盖根部署路径与带 wiki 前缀的部署路径。
func TestRestoreProgressDuringMaintenance(t *testing.T) {
	gate := restore.NewWriteGate()
	gate.Engage()
	router := newTestRouter(gate)
	for _, path := range []string{"/api/restore/progress", "/wiki/api/restore/progress"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(method, path, nil))
			expected := http.StatusOK
			if method == http.MethodPost {
				expected = http.StatusServiceUnavailable
			}
			if response.Code != expected {
				t.Fatalf("%s %s=%d want=%d", method, path, response.Code, expected)
			}
		}
	}
}
