// 本文件从公开方法验证任务状态转换与并发启动互斥，不执行真实索引重建。
package wikiresync_test

import (
	"sync"
	"testing"

	wikiresync "github.com/NianBroken/LeafDoc/internal/wiki/resync"
)

// TestResyncJob_NewJobIsIdle 核对空闲对象的四项初始字段，防止空闲被误判为结束。
func TestResyncJob_NewJobIsIdle(t *testing.T) {
	job := wikiresync.NewResyncJob()
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
	if s.Error != "" {
		t.Errorf("new job should have no error, got %q", s.Error)
	}
}

// TestResyncJob_StartSetsRunning 验证首次启动成功并进入运行中状态。
func TestResyncJob_StartSetsRunning(t *testing.T) {
	job := wikiresync.NewResyncJob()

	if !job.Start() {
		t.Fatal("Start() should return true for a new idle job")
	}

	s := job.Status()
	if !s.Running {
		t.Error("job should be running after Start()")
	}
	if s.Done {
		t.Error("job should not be done after Start()")
	}
}

// TestResyncJob_StartReturnsFalseWhenAlreadyRunning 验证运行占用拒绝第二次启动。
func TestResyncJob_StartReturnsFalseWhenAlreadyRunning(t *testing.T) {
	job := wikiresync.NewResyncJob()
	job.Start()

	if job.Start() {
		t.Error("Start() should return false when job is already running")
	}
}

// TestResyncJob_SetPhaseUpdatesPhase 验证工作过程写入的阶段通过状态副本读出。
func TestResyncJob_SetPhaseUpdatesPhase(t *testing.T) {
	job := wikiresync.NewResyncJob()
	job.Start()
	job.SetPhase(wikiresync.PhaseLinks)

	s := job.Status()
	if s.Phase != string(wikiresync.PhaseLinks) {
		t.Errorf("expected phase %q, got %q", wikiresync.PhaseLinks, s.Phase)
	}
}

// TestResyncJob_FinishSuccess 验证成功结束会释放运行占用、标记完成且不携带错误。
func TestResyncJob_FinishSuccess(t *testing.T) {
	job := wikiresync.NewResyncJob()
	job.Start()
	job.SetPhase(wikiresync.PhaseSearch)
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
}

// TestResyncJob_FinishWithError 验证失败也进入结束状态，并保留原始错误文字。
func TestResyncJob_FinishWithError(t *testing.T) {
	job := wikiresync.NewResyncJob()
	job.Start()
	job.SetPhase(wikiresync.PhaseTree)

	job.Finish(errTest("tree reconstruction failed"))

	s := job.Status()
	if s.Running {
		t.Error("job should not be running after Finish(err)")
	}
	if !s.Done {
		t.Error("job should be done after Finish(err)")
	}
	if s.Error != "tree reconstruction failed" {
		t.Errorf("expected error message, got %q", s.Error)
	}
}

// TestResyncJob_StartAfterFinishResetsState 验证结束后能够再次启动，
// 新一轮运行不继承上一轮的完成标记与阶段。
func TestResyncJob_StartAfterFinishResetsState(t *testing.T) {
	job := wikiresync.NewResyncJob()
	job.Start()
	job.SetPhase(wikiresync.PhaseSearch)
	job.Finish(nil)

	// 结束状态不占用任务，成功启动后读取的字段应属于这一轮运行。
	if !job.Start() {
		t.Fatal("Start() should return true after job has finished")
	}
	s := job.Status()
	if !s.Running {
		t.Error("job should be running after second Start()")
	}
	if s.Done {
		t.Error("job should not be done right after re-Start()")
	}
	if s.Phase != "" {
		t.Errorf("phase should be reset after re-Start(), got %q", s.Phase)
	}
}

// TestResyncJob_ConcurrentStartIsSafe 让十个协程竞争同一个任务，
// 各自只写结果数组中的独立位置，等待全部结束后统计唯一成功者。
func TestResyncJob_ConcurrentStartIsSafe(t *testing.T) {
	job := wikiresync.NewResyncJob()

	var wg sync.WaitGroup
	started := make([]bool, 10)
	for i := range started {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			started[idx] = job.Start()
		}(i)
	}
	wg.Wait()

	count := 0
	for _, ok := range started {
		if ok {
			count++
		}
	}
	if count != 1 {
		t.Errorf("exactly one goroutine should win Start(), got %d", count)
	}
}

// errTest 将固定文字包装成错误，使断言直接比较状态返回的错误内容。
type errTest string

// Error 原样返回测试错误文字，不添加前缀或其他上下文。
func (e errTest) Error() string { return string(e) }
