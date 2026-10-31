package importer

import (
	"errors"
	"testing"
	"time"
)

// TestPlanStoreSet 验证内存状态可被读取，返回外层副本而非原对象指针。
func TestPlanStoreSet(t *testing.T) {
	s := NewPlanStore()
	plan := &StoredPlan{ExecutionStatus: ExecutionStatusPlanned}
	if err := s.Set(plan); err != nil {
		t.Fatalf("Set err: %v", err)
	}

	retrieved, err := s.Get()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retrieved == plan {
		t.Fatalf("expected Get to return a snapshot copy")
	}
	if retrieved.ExecutionStatus != plan.ExecutionStatus {
		t.Fatalf("expected execution status %q, got %q", plan.ExecutionStatus, retrieved.ExecutionStatus)
	}
}

// TestPlanStoreGet 确认空状态库不能返回有效计划；此断言只检查错误非空。
func TestPlanStoreGet(t *testing.T) {
	s := NewPlanStore()

	_, err := s.Get()
	if err == nil {
		t.Fatalf("expected error when getting plan from empty store")
	}
}

// TestPlanStoreSetAndGet 验证计划标识随状态保存，同时维持外层对象隔离。
func TestPlanStoreSetAndGet(t *testing.T) {
	s := NewPlanStore()
	plan := &StoredPlan{
		Plan:            &PlanResult{ID: "plan-1"},
		ExecutionStatus: ExecutionStatusPlanned,
	}
	if err := s.Set(plan); err != nil {
		t.Fatalf("Set err: %v", err)
	}

	retrieved, err := s.Get()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retrieved == plan {
		t.Fatalf("expected Get to return a snapshot copy")
	}
	if retrieved.Plan == nil || retrieved.Plan.ID != "plan-1" {
		t.Fatalf("expected retrieved plan ID to match, got %#v", retrieved.Plan)
	}
}

// TestPlanStoreClear 在写入后清理状态，再读取必须失败，测试不使用数据库持久化。
func TestPlanStoreClear(t *testing.T) {
	s := NewPlanStore()
	plan := &StoredPlan{}
	if err := s.Set(plan); err != nil {
		t.Fatalf("Set err: %v", err)
	}

	if _, err := s.Clear(); err != nil {
		t.Fatalf("Clear err: %v", err)
	}

	_, err := s.Get()
	if err == nil {
		t.Fatalf("expected error when getting plan from cleared store")
	}
}

// TestPlanStoreTryStartExecution_WithNilPlanPayload 区分存在状态对象与存在有效计划。
// 内部 Plan 为空时启动返回状态不可用错误，不将其当作正常待执行计划。
func TestPlanStoreTryStartExecution_WithNilPlanPayload(t *testing.T) {
	store := NewPlanStore()
	if err := store.Set(&StoredPlan{
		Plan:            nil,
		ExecutionStatus: ExecutionStatusPlanned,
	}); err != nil {
		t.Fatalf("Set err: %v", err)
	}

	_, _, err := store.TryStartExecution("user-1")
	if !errors.Is(err, ErrImportStateUnavailable) {
		t.Fatalf("expected ErrImportStateUnavailable, got %v", err)
	}
}

// TestPlanStoreUpdateExecutionProgress_UpdatesEmbeddedFields 验证进度写入嵌入字段。
// 断言覆盖处理数、总数、源路径和开始时间，不依赖 JSON 序列化或数据库。
func TestPlanStoreUpdateExecutionProgress_UpdatesEmbeddedFields(t *testing.T) {
	store := NewPlanStore()
	if err := store.Set(&StoredPlan{
		Plan:            &PlanResult{ID: "plan-1"},
		ExecutionStatus: ExecutionStatusRunning,
	}); err != nil {
		t.Fatalf("Set err: %v", err)
	}

	now := time.Now()
	sourcePath := "docs/readme.md"
	err := store.UpdateExecutionProgress("plan-1", ExecutionProgress{
		ProcessedItems:        2,
		TotalItems:            5,
		CurrentItemSourcePath: &sourcePath,
		StartedAt:             &now,
	}, nil)
	if err != nil {
		t.Fatalf("UpdateExecutionProgress err: %v", err)
	}

	retrieved, err := store.Get()
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if retrieved.ProcessedItems != 2 || retrieved.TotalItems != 5 {
		t.Fatalf("unexpected progress values: %#v", retrieved.ExecutionProgress)
	}
	if retrieved.CurrentItemSourcePath == nil || *retrieved.CurrentItemSourcePath != sourcePath {
		t.Fatalf("unexpected current item source path: %#v", retrieved.CurrentItemSourcePath)
	}
	if retrieved.StartedAt == nil || !retrieved.StartedAt.Equal(now) {
		t.Fatalf("expected started_at to be updated, got %#v", retrieved.StartedAt)
	}
}
