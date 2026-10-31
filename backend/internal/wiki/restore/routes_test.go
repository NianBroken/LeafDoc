// 本文件直接测试空恢复管理器的拒绝响应，不挂载管理员认证中间件。
package wikirestore

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// init 让最小测试路由在 Gin 测试模式运行。
func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter 注册真实恢复处理器，保持参数路径与进度入口的解析方式一致。
func newTestRouter(routes *Routes) *gin.Engine {
	r := gin.New()
	r.POST("/restore/upload", routes.handleTriggerUpload)
	r.POST("/restore/:id", routes.handleTrigger)
	r.GET("/restore/status", routes.handleStatus)
	r.GET("/api/restore/progress", routes.handleProgress)
	r.POST("/restore/self-restart", routes.handleSelfRestart)
	return r
}

// TestHandleTrigger_NotEnabledWhenManagerNil 验证已有快照恢复先检查管理器依赖。
func TestHandleTrigger_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/restore/snapshot-20260101-000000", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// TestHandleStatus_NotEnabledWhenManagerNil 验证状态查询对未启用功能返回 503。
func TestHandleStatus_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/restore/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// TestHandleSelfRestart_NotEnabledWhenManagerNil 验证无管理器时不会调用平台重启。
func TestHandleSelfRestart_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/restore/self-restart", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
