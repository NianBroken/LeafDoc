// 本文件直接验证恢复任务状态对象，不执行实际文件交换或数据库恢复。
package restore

import "testing"

// 新任务不运行、不完成、无阶段且无需介入，供管理器识别初始状态。
func TestJob_NewJobIsIdle(t *testing.T) {
	job := NewJob()
	s := job.Status()

	if s.Running {
		t.Error("new job should not be running")
	}
	if s.Done {
		t.Error("new job should not be done")
	}
	if s.Phase != "" {
		t.Errorf("new job should have no phase, got %q", s.Phase)
	}
	if s.NeedsIntervention {
		t.Error("new job should not need intervention")
	}
}

// 空闲任务成功开始后，状态副本必须显示正在运行。
func TestJob_StartSetsRunning(t *testing.T) {
	job := NewJob()

	if !job.Start() {
		t.Fatal("Start() should return true for a new idle job")
	}
	s := job.Status()
	if !s.Running {
		t.Error("job should be running after Start()")
	}
}

// 连续开始同一运行任务应被第二次调用拒绝，不另建任务状态。
func TestJob_StartReturnsFalseWhenAlreadyRunning(t *testing.T) {
	job := NewJob()
	job.Start()

	if job.Start() {
		t.Error("Start() should return false when job is already running")
	}
}

// 设置交换阶段后，状态接口返回对应字符串，不靠实际交换资源推进阶段。
func TestJob_SetPhaseUpdatesPhase(t *testing.T) {
	job := NewJob()
	job.Start()
	job.SetPhase(PhaseSwapping)

	if s := job.Status(); s.Phase != string(PhaseSwapping) {
		t.Errorf("expected phase %q, got %q", PhaseSwapping, s.Phase)
	}
}

// 从普通运行态成功结束后，运行标记清空、完成标记置位且没有错误或介入标记。
func TestJob_FinishSuccess(t *testing.T) {
	job := NewJob()
	job.Start()
	job.Finish(nil)

	s := job.Status()
	if s.Running {
		t.Error("job should not be running after Finish()")
	}
	if !s.Done {
		t.Error("job should be done after Finish()")
	}
	if s.Error != "" {
		t.Errorf("expected no error, got %q", s.Error)
	}
	if s.NeedsIntervention {
		t.Error("Finish(nil) should not set NeedsIntervention")
	}
}

// 需要介入的结束入口保存错误文本，同时结束运行并设置介入标记。
func TestJob_FinishNeedsIntervention(t *testing.T) {
	job := NewJob()
	job.Start()
	job.FinishNeedsIntervention(errTest("rollback also failed"))

	s := job.Status()
	if s.Running {
		t.Error("job should not be running after FinishNeedsIntervention()")
	}
	if !s.Done {
		t.Error("job should be done after FinishNeedsIntervention()")
	}
	if !s.NeedsIntervention {
		t.Error("expected NeedsIntervention = true")
	}
	if s.Error != "rollback also failed" {
		t.Errorf("expected error message, got %q", s.Error)
	}
}

// Job 自身允许已结束任务再次开始并清空介入标记，
// 管理器拒绝介入状态的新任务属于上层规则，不由这个对象执行。
func TestJob_StartAfterFinishNeedsInterventionResetsFlag(t *testing.T) {
	job := NewJob()
	job.Start()
	job.FinishNeedsIntervention(errTest("boom"))

	if !job.Start() {
		t.Fatal("expected Start() to succeed again after a finished job")
	}
	if s := job.Status(); s.NeedsIntervention {
		t.Error("expected NeedsIntervention to reset on a fresh Start()")
	}
}

// 非致命版本提示能通过状态接口读到，不需要结束任务。
func TestJob_SetVersionWarning(t *testing.T) {
	job := NewJob()
	job.Start()
	job.SetVersionWarning("snapshot was created by v0.9.0, this server is running v0.10.0")

	if s := job.Status(); s.VersionWarning == "" {
		t.Error("expected version warning to be set")
	}
}

// errTest 将固定测试文本实现为 error，便于精确检查状态中的错误内容。
type errTest string

func (e errTest) Error() string { return string(e) }
