// 本文件验证非法归档拒绝、目录交换失败回滚、并发任务拒绝和写入等待超时。
// 测试使用临时文件与注入失败，不执行真实数据库内容替换。
package restore

import (
	"archive/zip"

	"errors"

	"os"
	"path/filepath"
	"testing"
	"time"
)

// 两种归档都含三个必要入口，但使用无效空元数据与摘要，其中一项还含穿越路径。
// 断言归档必须被拒绝，不限定失败来自必要项、摘要还是路径检查。
func TestRejectLegacyAndUnsafeArchive(t *testing.T) {
	for _, name := range []string{"legacy", "../escape.txt"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			file, err := os.Create(filepath.Join(dir, "bad.zip"))
			if err != nil {
				t.Fatal(err)
			}
			writer := zip.NewWriter(file)
			for _, entry := range []string{"database.json", "backup-meta.json", "checksums.json", name} {
				target, err := writer.Create(entry)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := target.Write([]byte("{}")); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if staging, _, err := extractAndValidate(file.Name(), dir); err == nil {
				if err := os.RemoveAll(staging); err != nil {
					t.Fatal(err)
				}
				t.Fatal("unsafe or invalid archive accepted")
			}
		})
	}
}

// 在旧目录移走后让第二次改名失败，验证回滚仍能识别 movedAside，
// 把原文件及其内容放回当前目录，不只处理已经成功移入的项。
func TestResourceRollback(t *testing.T) {
	dir := t.TempDir()
	staging := t.TempDir()
	for _, base := range []string{dir, staging} {
		if err := os.MkdirAll(filepath.Join(base, "assets"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	original := filepath.Join(dir, "assets", "original.txt")
	if err := os.WriteFile(original, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	swap := newSwapper(dir, staging)
	calls := 0
	// 仅第二次改名返回权限错误，其他改名执行真实临时目录操作。
	swap.rename = func(from, to string) error {
		calls++
		if calls == 2 {
			return os.ErrPermission
		}
		return os.Rename(from, to)
	}
	if err := swap.SwapAll(); err == nil {
		t.Fatal("injected failure not propagated")
	}
	if err := swap.RollbackAll(); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(original); err != nil || string(raw) != "original" {
		t.Fatal("resource rollback lost original")
	}
}

// 直接占用任务状态后，管理器必须在查找快照前拒绝另一个恢复请求。
func TestRestoreRejectsConcurrentJob(t *testing.T) {
	manager := NewManager(Config{})
	manager.job.Start()
	if !errors.Is(manager.TriggerRestore("anything"), ErrRestoreAlreadyRunning) {
		t.Fatal("concurrent restore accepted")
	}
}

// 保留一项在途登记并关闭准入，短超时应返回失败。
// 用例没有磁盘数据，因此只验证屏障超时，不验证恢复管理器的文件保留流程。
func TestWriteDrainTimeoutLeavesData(t *testing.T) {
	gate := NewWriteGate()
	leave, ok := gate.TryEnter()
	if !ok {
		t.Fatal("write refused")
	}
	defer leave()
	gate.Engage()
	if gate.WaitForDrain(time.Millisecond) {
		t.Fatal("inflight write ignored")
	}
}
