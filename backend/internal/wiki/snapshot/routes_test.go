// 本文件直接挂载快照处理器，验证缺失依赖时的响应，不覆盖管理员认证链。
package wikisnapshot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// init 使测试路由使用 Gin 测试模式，避免输出调试路由注册信息。
func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter 挂载六种真实处理方法，测试用空 Routes 构造未启用状态。
func newTestRouter(routes *Routes) *gin.Engine {
	r := gin.New()
	r.GET("/snapshot/status", routes.handleStatus)
	r.PUT("/snapshot/settings", routes.handleUpdateSettings)
	r.GET("/snapshot", routes.handleList)
	r.POST("/snapshot", routes.handleTrigger)
	r.GET("/snapshot/:id/download", routes.handleDownload)
	r.DELETE("/snapshot/:id", routes.handleDelete)
	return r
}

// TestHandleStatus_Disabled 验证没有管理器时 JSON 明确报告 enabled=false。
func TestHandleStatus_Disabled(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/snapshot/status", nil)
	router.ServeHTTP(w, req)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["enabled"] != false {
		t.Errorf("expected enabled=false, got %v", body)
	}
}

// TestHandleList_NotEnabledWhenManagerNil 验证列表入口不访问空管理器而返回 503。
func TestHandleList_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/snapshot", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// TestHandleDownload_NotEnabledWhenManagerNil 验证下载在解析归档路径前检查依赖。
func TestHandleDownload_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/snapshot/snapshot-20260101-000000/download", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// TestHandleDelete_NotEnabledWhenManagerNil 验证删除缺少管理器时返回服务不可用。
func TestHandleDelete_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/snapshot/snapshot-20260101-000000", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// TestHandleTrigger_NotEnabledWhenSchedulerNil 验证手动排队需要调度器而非仅注册路由。
func TestHandleTrigger_NotEnabledWhenSchedulerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/snapshot", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
