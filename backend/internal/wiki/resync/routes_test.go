// 本文件与处理器使用同一个包，直接注册真实处理方法，验证 HTTP 状态与 JSON 内容。
// 最小路由不安装身份认证中间件，因此这些测试不覆盖管理员权限校验。
package wikiresync

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newTestRouter 将真实用例和处理器接到独立 Gin 引擎，启动回调由各场景提供。
func newTestRouter(job *ResyncJob, trigger func()) *gin.Engine {
	r := gin.New()
	routes := &Routes{
		triggerUC: NewTriggerResyncUseCase(job, trigger, nil),
		statusUC:  NewGetResyncStatusUseCase(job),
	}
	r.POST("/resync", routes.handleTriggerResync)
	r.GET("/resync/status", routes.handleResyncStatus)
	return r
}

// TestHandleTriggerResync_Returns202 验证接受请求的状态、ok 字段与回调实际执行。
// 带缓冲通道允许同步回调写入而不等待测试读取。
func TestHandleTriggerResync_Returns202(t *testing.T) {
	job := NewResyncJob()
	launched := make(chan struct{}, 1)
	router := newTestRouter(job, func() {
		launched <- struct{}{}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/resync", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["ok"] != true {
		t.Errorf("expected {ok:true}, got %v", body)
	}
	select {
	case <-launched:
	default:
		t.Error("trigger was not called")
	}
}

// TestHandleTriggerResync_Returns409WhenAlreadyRunning 验证冲突响应包含业务错误码，
// 并且任务已占用时不会执行传入的启动回调。
func TestHandleTriggerResync_Returns409WhenAlreadyRunning(t *testing.T) {
	job := NewResyncJob()
	job.Start() // 直接占用任务，构造已有工作正在执行的状态。

	called := false
	router := newTestRouter(job, func() { called = true })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/resync", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
	var body ResyncErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Error.Code != ErrCodeResyncAlreadyRunning {
		t.Errorf("expected code %q, got %q", ErrCodeResyncAlreadyRunning, body.Error.Code)
	}
	if called {
		t.Error("trigger must not be called when job is already running")
	}
}

// TestHandleResyncStatus_IdleState 核对尚未执行任务时接口返回 200 和空闲标记。
func TestHandleResyncStatus_IdleState(t *testing.T) {
	job := NewResyncJob()
	router := newTestRouter(job, func() {})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/resync/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var s JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if s.Running {
		t.Error("idle job should not be running")
	}
	if s.Done {
		t.Error("idle job should not be done")
	}
}

// TestHandleResyncStatus_RunningState 验证运行标记与链接阶段经 JSON 响应保留。
func TestHandleResyncStatus_RunningState(t *testing.T) {
	job := NewResyncJob()
	job.Start()
	job.SetPhase(PhaseLinks)

	router := newTestRouter(job, func() {})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/resync/status", nil)
	router.ServeHTTP(w, req)

	var s JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !s.Running {
		t.Error("job should be running")
	}
	if s.Phase != "links" {
		t.Errorf("expected phase 'links', got %q", s.Phase)
	}
}

// TestHandleResyncStatus_DoneState 验证成功结束的状态不再运行且没有错误文字。
func TestHandleResyncStatus_DoneState(t *testing.T) {
	job := NewResyncJob()
	job.Start()
	job.SetPhase(PhaseSearch)
	job.Finish(nil)

	router := newTestRouter(job, func() {})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/resync/status", nil)
	router.ServeHTTP(w, req)

	var s JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if s.Running {
		t.Error("finished job should not be running")
	}
	if !s.Done {
		t.Error("finished job should be done")
	}
	if s.Error != "" {
		t.Errorf("expected no error, got %q", s.Error)
	}
}

// TestHandleResyncStatus_DoneWithError 验证工作错误作为状态内容返回，不丢失失败原因。
func TestHandleResyncStatus_DoneWithError(t *testing.T) {
	job := NewResyncJob()
	job.Start()
	job.SetPhase(PhaseTree)
	job.Finish(errors.New("tree failed"))

	router := newTestRouter(job, func() {})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/resync/status", nil)
	router.ServeHTTP(w, req)

	var s JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !s.Done {
		t.Error("expected done=true")
	}
	if s.Error != "tree failed" {
		t.Errorf("expected error message, got %q", s.Error)
	}
}

// TestHandleTriggerResync_PhaseOmittedWhenEmpty 实际查询状态接口，
// 通过原始 JSON 映射确认空阶段整个字段被省略，而非写成空字符串。
func TestHandleTriggerResync_PhaseOmittedWhenEmpty(t *testing.T) {
	job := NewResyncJob()
	router := newTestRouter(job, func() {})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/resync/status", nil)
	router.ServeHTTP(w, req)

	// omitempty 使空阶段缺席，前端读取缺席字段后可用空值合并得到 null。
	var raw map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := raw["phase"]; ok {
		t.Errorf("phase key must be absent when empty, got %v", raw["phase"])
	}
}
