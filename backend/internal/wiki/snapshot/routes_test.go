package wikisnapshot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

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

func TestHandleList_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/snapshot", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleDownload_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/snapshot/snapshot-20260101-000000/download", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleDelete_NotEnabledWhenManagerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/snapshot/snapshot-20260101-000000", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleTrigger_NotEnabledWhenSchedulerNil(t *testing.T) {
	router := newTestRouter(&Routes{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/snapshot", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
