// 本文件在临时备份目录验证列表、删除、保留数量和标识约束。
// 样本使用旁置 JSON 和占位 ZIP，不执行数据库导出或证明压缩包内容有效。
package snapshot

import (
	"encoding/json"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"

	"os"
	"path/filepath"
	"testing"
	"time"
)

// 仅创建两份元数据，确认列表按记录时间从新到旧排列，不要求 ZIP 同时存在。
func TestManager_List(t *testing.T) {
	backupsDir := t.TempDir()
	m := NewManager(Config{BackupsDir: backupsDir})

	older := SnapshotEntry{ID: "snapshot-20260101-000000", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), SizeBytes: 100}
	newer := SnapshotEntry{ID: "snapshot-20260201-000000", CreatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), SizeBytes: 200}

	writeSidecarFixture(t, backupsDir, older)
	writeSidecarFixture(t, backupsDir, newer)

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List returned %d entries, want 2", len(entries))
	}
	if entries[0].ID != newer.ID || entries[1].ID != older.ID {
		t.Errorf("List order = [%s, %s], want [%s, %s]", entries[0].ID, entries[1].ID, newer.ID, older.ID)
	}
}

// 备份目录尚未创建时，列表返回空结果而不是文件系统错误。
func TestManager_List_EmptyWhenBackupsDirMissing(t *testing.T) {
	m := NewManager(Config{BackupsDir: filepath.Join(t.TempDir(), "does-not-exist")})

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List returned %d entries, want 0", len(entries))
	}
}

// writeSidecarFixture 创建目录并按生产字段格式写入单份 JSON，不生成归档。
// Helper 使失败位置归到调用用例，任一步失败立即结束当前测试。
func writeSidecarFixture(t *testing.T, backupsDir string, entry SnapshotEntry) {
	t.Helper()
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatalf("failed to create backups dir: %v", err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal fixture: %v", err)
	}
	path := filepath.Join(backupsDir, entry.ID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
}

// 合法标识的 ZIP 和 JSON 都缺失时，删除返回未找到的本地化错误代码。
func TestManager_Delete_NotFound(t *testing.T) {
	backupsDir := t.TempDir()
	m := NewManager(Config{BackupsDir: backupsDir})

	err := m.Delete("snapshot-20260101-000000")
	assertLocalizedErrorCode(t, err, "snapshot_not_found")
}

// 两类文件存在时均应被删除；ZIP 内容只是占位字节，删除入口不解析归档。
func TestManager_Delete_RemovesFiles(t *testing.T) {
	backupsDir := t.TempDir()
	m := NewManager(Config{BackupsDir: backupsDir})

	entry := SnapshotEntry{ID: "snapshot-20260101-000000", CreatedAt: time.Now().UTC(), SizeBytes: 42}
	writeSidecarFixture(t, backupsDir, entry)
	zipPath := filepath.Join(backupsDir, entry.ID+".zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip"), 0o644); err != nil {
		t.Fatalf("failed to write fixture zip: %v", err)
	}

	if err := m.Delete(entry.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
		t.Errorf("expected zip file to be removed")
	}
	if _, err := os.Stat(filepath.Join(backupsDir, entry.ID+".json")); !os.IsNotExist(err) {
		t.Errorf("expected sidecar file to be removed")
	}
}

// 三份按日递增的元数据只保留最新两份，验证清理使用元数据时间而非文件写入时间。
func TestManager_PruneOldSnapshots_KeepsNewestOnly(t *testing.T) {
	backupsDir := t.TempDir()
	m := NewManager(Config{BackupsDir: backupsDir, RetentionCount: 2})

	entries := []SnapshotEntry{
		{ID: "snapshot-20260101-000000", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), SizeBytes: 1},
		{ID: "snapshot-20260102-000000", CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), SizeBytes: 1},
		{ID: "snapshot-20260103-000000", CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), SizeBytes: 1},
	}
	for _, e := range entries {
		writeSidecarFixture(t, backupsDir, e)
	}

	if err := m.pruneOldSnapshots(); err != nil {
		t.Fatalf("pruneOldSnapshots failed: %v", err)
	}

	remaining, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining snapshots, got %d", len(remaining))
	}
	want := map[string]bool{"snapshot-20260103-000000": true, "snapshot-20260102-000000": true}
	for _, r := range remaining {
		if !want[r.ID] {
			t.Errorf("unexpected snapshot retained: %s", r.ID)
		}
	}
}

// 保留数为零时跳过清理，两份元数据均保留。此用例没有单独传入负保留数。
func TestManager_PruneOldSnapshots_UnlimitedWhenRetentionNotPositive(t *testing.T) {
	backupsDir := t.TempDir()
	m := NewManager(Config{BackupsDir: backupsDir, RetentionCount: 0})

	writeSidecarFixture(t, backupsDir, SnapshotEntry{ID: "snapshot-20260101-000000", CreatedAt: time.Now().UTC(), SizeBytes: 1})
	writeSidecarFixture(t, backupsDir, SnapshotEntry{ID: "snapshot-20260102-000000", CreatedAt: time.Now().UTC(), SizeBytes: 1})

	if err := m.pruneOldSnapshots(); err != nil {
		t.Fatalf("pruneOldSnapshots failed: %v", err)
	}

	remaining, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(remaining) != 2 {
		t.Errorf("expected no pruning with RetentionCount=0, got %d remaining", len(remaining))
	}
}

// 路径穿越、缺少固定开头和含连续双点的标识都在文件操作前被拒绝。
func TestManager_Delete_InvalidID(t *testing.T) {
	backupsDir := t.TempDir()
	m := NewManager(Config{BackupsDir: backupsDir})

	for _, id := range []string{"../etc/passwd", "not-a-snapshot", "snapshot-../evil"} {
		assertLocalizedErrorCode(t, m.Delete(id), "snapshot_invalid_id")
	}
}

// assertLocalizedErrorCode 同时确认错误类型和代码，不依赖界面语言或详细错误文本。
func assertLocalizedErrorCode(t *testing.T, err error, wantCode string) {
	t.Helper()
	localized, ok := sharederrors.AsLocalizedError(err)
	if !ok {
		t.Fatalf("expected localized error, got %T (%v)", err, err)
	}
	if localized.Code != wantCode {
		t.Fatalf("localized error code = %q, want %q", localized.Code, wantCode)
	}
}
