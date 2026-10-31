// 本文件验证准入开关、在途计数和轮询等待。测试成功登记后均调用离开函数，
// 并发用例通过独立下标记录结果，等待全部协程后才检查。
package restore

import (
	"sync"
	"testing"
	"time"
)

// 默认屏障开放，成功登记并离开不改变准入开关。
func TestWriteGate_NewGateAllowsEntry(t *testing.T) {
	g := NewWriteGate()

	leave, ok := g.TryEnter()
	if !ok {
		t.Fatal("expected TryEnter to succeed on a fresh gate")
	}
	leave()

	if g.Engaged() {
		t.Error("fresh gate should not be engaged")
	}
}

// 关闭准入后状态必须为关闭，新登记立即失败。
func TestWriteGate_EngageBlocksEntry(t *testing.T) {
	g := NewWriteGate()
	g.Engage()

	if !g.Engaged() {
		t.Error("expected gate to be engaged")
	}
	if _, ok := g.TryEnter(); ok {
		t.Error("expected TryEnter to fail while engaged")
	}
}

// 重新开放屏障后可以再次登记，返回的离开函数仍须执行。
func TestWriteGate_DisengageAllowsEntryAgain(t *testing.T) {
	g := NewWriteGate()
	g.Engage()
	g.Disengage()

	if g.Engaged() {
		t.Error("expected gate to be disengaged")
	}
	leave, ok := g.TryEnter()
	if !ok {
		t.Fatal("expected TryEnter to succeed after Disengage")
	}
	leave()
}

// 没有在途请求时无需等待完整超时，允许少量调度开销但要求在半秒内返回。
func TestWriteGate_WaitForDrain_ReturnsImmediatelyWhenEmpty(t *testing.T) {
	g := NewWriteGate()

	start := time.Now()
	if !g.WaitForDrain(2 * time.Second) {
		t.Fatal("expected WaitForDrain to succeed with no in-flight requests")
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("expected WaitForDrain to return promptly, took %v", elapsed)
	}
}

// 后台协程延迟离开，等待入口应观察到计数归零后成功，完成通道确保协程退出。
func TestWriteGate_WaitForDrain_WaitsForInFlightRequestsToLeave(t *testing.T) {
	g := NewWriteGate()

	leave, ok := g.TryEnter()
	if !ok {
		t.Fatal("expected TryEnter to succeed")
	}

	done := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		leave()
		close(done)
	}()

	if !g.WaitForDrain(2 * time.Second) {
		t.Fatal("expected WaitForDrain to succeed once the in-flight request left")
	}
	<-done
}

// 在等待期间保持一项登记，要求等待失败且耗时不短于指定超时，
// 随后通过延迟离开恢复计数，测试不把超时当成自动撤销登记。
func TestWriteGate_WaitForDrain_TimesOutWhenRequestNeverLeaves(t *testing.T) {
	g := NewWriteGate()

	leave, ok := g.TryEnter()
	if !ok {
		t.Fatal("expected TryEnter to succeed")
	}
	defer leave()

	start := time.Now()
	if g.WaitForDrain(150 * time.Millisecond) {
		t.Fatal("expected WaitForDrain to time out while a request is still in flight")
	}
	if elapsed := time.Since(start); elapsed < 150*time.Millisecond {
		t.Errorf("expected WaitForDrain to wait out the timeout, returned after %v", elapsed)
	}
}

func TestWriteGate_EngageAfterTryEnterStillLetsInFlightRequestFinish(t *testing.T) {
	// 关闭准入只拒绝后续登记，已有请求的离开函数仍可递减计数。
	g := NewWriteGate()

	leave, ok := g.TryEnter()
	if !ok {
		t.Fatal("expected TryEnter to succeed before Engage")
	}
	g.Engage()

	leave()
	if !g.WaitForDrain(time.Second) {
		t.Fatal("expected drain to succeed after the pre-Engage request left")
	}
}

// 五十个协程在开放状态并发登记和离开，每项都应获得准入，不制造关闭竞争。
func TestWriteGate_ConcurrentTryEnterIsSafe(t *testing.T) {
	g := NewWriteGate()

	var wg sync.WaitGroup
	admitted := make([]bool, 50)
	for i := range admitted {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			leave, ok := g.TryEnter()
			admitted[idx] = ok
			if ok {
				leave()
			}
		}(i)
	}
	wg.Wait()

	for i, ok := range admitted {
		if !ok {
			t.Errorf("goroutine %d expected TryEnter to succeed on a disengaged gate", i)
		}
	}
}
