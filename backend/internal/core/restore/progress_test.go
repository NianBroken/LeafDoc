// 本文件直接驱动任务开始与结束，验证恢复进度凭据的独立生命周期。
// 不启动归档恢复，也不创建真实账号会话。
package restore

import (
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

// 未签发、错误、旧任务、其他管理器及已到期凭据均不能读取状态。
// 当前任务完成仍可凭原令牌查询，新任务成功开始后轮换令牌，并发启动被拒绝时保留原令牌。
func TestProgressCredentialLifecycle(t *testing.T) {
	manager := NewManager(Config{})
	if _, ok := manager.ProgressStatus(""); ok {
		t.Fatal("idle manager authorized anonymous progress")
	}
	manager.progressMu.Lock()
	first, err := manager.startProgressJob()
	manager.progressMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	// 解码长度确认令牌承载三十二字节，不能仅凭长度断言其随机分布。
	random, err := hex.DecodeString(first)
	if err != nil || len(random) != 32 {
		t.Fatal("credential is not 256 random bits")
	}
	if _, ok := manager.ProgressStatus(first); !ok {
		t.Fatal("current credential rejected")
	}
	if _, ok := manager.ProgressStatus("wrong"); ok {
		t.Fatal("incorrect credential accepted")
	}
	manager.progressMu.Lock()
	_, err = manager.startProgressJob()
	manager.progressMu.Unlock()
	if !errors.Is(err, ErrRestoreAlreadyRunning) {
		t.Fatalf("concurrent start: %v", err)
	}
	if _, ok := manager.ProgressStatus(first); !ok {
		t.Fatal("rejected concurrent job invalidated active credential")
	}
	// 直接完成状态模拟后台任务结束，凭据不因任务完成而立即失效。
	manager.job.Finish(nil)
	if status, ok := manager.ProgressStatus(first); !ok || !status.Done {
		t.Fatal("completed job inaccessible after session invalidation")
	}
	manager.progressMu.Lock()
	second, err := manager.startProgressJob()
	manager.progressMu.Unlock()
	if err != nil || second == first {
		t.Fatal("new job failed to rotate credential")
	}
	if _, ok := manager.ProgressStatus(first); ok {
		t.Fatal("previous job credential accepted")
	}
	if _, ok := manager.ProgressStatus(second); !ok {
		t.Fatal("new credential rejected")
	}
	if _, ok := NewManager(Config{}).ProgressStatus(second); ok {
		t.Fatal("credential survived manager restart")
	}
	manager.progressMu.Lock()
	manager.progressExpiresAt = time.Now().Add(-time.Second)
	manager.progressMu.Unlock()
	if _, ok := manager.ProgressStatus(second); ok {
		t.Fatal("expired progress credential accepted")
	}

	manager.job.Finish(nil)
}
