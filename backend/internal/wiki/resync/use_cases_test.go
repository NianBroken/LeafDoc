// 本文件验证触发用例的同步回调、重复任务拒绝以及状态查询结果。
package wikiresync_test

import (
	"context"
	"errors"
	"testing"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	. "github.com/NianBroken/LeafDoc/internal/wiki/resync"
)

// TestTriggerResyncUseCase_Execute_LaunchesTrigger 通过回调写入标记，
// 确认 Execute 返回前已经调用启动函数，而非只改变任务状态。
func TestTriggerResyncUseCase_Execute_LaunchesTrigger(t *testing.T) {
	job := NewResyncJob()
	called := false
	uc := NewTriggerResyncUseCase(job, func() { called = true }, nil)

	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Error("trigger was not called")
	}
}

// TestTriggerResyncUseCase_Execute_ReturnsLocalizedErrorWhenAlreadyRunning
// 验证重复触发返回可翻译的冲突错误，同时禁止执行回调。
func TestTriggerResyncUseCase_Execute_ReturnsLocalizedErrorWhenAlreadyRunning(t *testing.T) {
	job := NewResyncJob()
	job.Start() // 预先占用任务，执行用例时必须走拒绝分支。

	uc := NewTriggerResyncUseCase(job, func() { t.Error("trigger must not be called") }, nil)
	err := uc.Execute(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	loc, ok := sharederrors.AsLocalizedError(err)
	if !ok {
		t.Fatalf("expected LocalizedError, got %T", err)
	}
	if loc.Code != ErrCodeResyncAlreadyRunning {
		t.Errorf("expected code %q, got %q", ErrCodeResyncAlreadyRunning, loc.Code)
	}
}

// TestGetResyncStatusUseCase_Execute_ReturnsJobStatus 验证查询保留结束标记与错误文字，
// 不因经过用例输出结构而丢失任务失败信息。
func TestGetResyncStatusUseCase_Execute_ReturnsJobStatus(t *testing.T) {
	job := NewResyncJob()
	job.Start()
	job.SetPhase(PhaseTags)
	job.Finish(errors.New("something went wrong"))

	uc := NewGetResyncStatusUseCase(job)
	out := uc.Execute(context.Background())

	if out.Status.Running {
		t.Error("finished job should not be running")
	}
	if !out.Status.Done {
		t.Error("expected done=true")
	}
	if out.Status.Error != "something went wrong" {
		t.Errorf("unexpected error message: %q", out.Status.Error)
	}
}
