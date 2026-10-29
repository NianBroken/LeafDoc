package snapshot

import (
	"archive/zip"
	"context"
	"encoding/json"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "snapshot") }

// newTestConfig supplies real MySQL state and independent resource paths.
func newTestConfig(t *testing.T) Config {
	t.Helper()
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	return Config{DataDir: dir, BackupsDir: filepath.Join(dir, "snapshots"), AssetsDir: filepath.Join(dir, "assets"), WikiVersion: "test"}
}

func TestMySQLSnapshotContents(t *testing.T) {
	cfg := newTestConfig(t)
	if err := database.WriteRecord(cfg.DataDir, "settings", "site", []byte("中文设置")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.AssetsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.AssetsDir, "sample.txt"), []byte("附件"), 0644); err != nil {
		t.Fatal(err)
	}
	id, err := createSnapshot(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(filepath.Join(cfg.BackupsDir, id+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	names := map[string]bool{}
	for _, entry := range reader.File {
		names[entry.Name] = true
	}
	for _, name := range []string{"database.json", "backup-meta.json", "checksums.json", "assets/sample.txt"} {
		if !names[name] {
			t.Errorf("missing %s", name)
		}
	}
	for _, entry := range reader.File {
		if filepath.Ext(entry.Name) == ".db" {
			t.Fatal("local database in MySQL snapshot")
		}
	}
}

func TestSnapshotWriteBarrierFailure(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.Quiesce = func() (func(), error) { return nil, os.ErrPermission }
	if _, err := createSnapshot(context.Background(), cfg); err == nil {
		t.Fatal("write barrier failure ignored")
	}
}

func TestManager_ErrAlreadyRunning(t *testing.T) {
	cfg := newTestConfig(t)
	m := NewManager(cfg)

	if err := m.TriggerSnapshot(); err != nil {
		t.Fatalf("first TriggerSnapshot failed: %v", err)
	}

	assertLocalizedErrorCode(t, m.TriggerSnapshot(), "snapshot_already_running")

	waitForSnapshotDone(t, m)
}

func waitForSnapshotDone(t *testing.T, m *Manager) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !m.Status().IsRunning {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("snapshot did not finish within deadline")
}

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

func TestManager_Delete_NotFound(t *testing.T) {
	backupsDir := t.TempDir()
	m := NewManager(Config{BackupsDir: backupsDir})

	err := m.Delete("snapshot-20260101-000000")
	assertLocalizedErrorCode(t, err, "snapshot_not_found")
}

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

func TestManager_RunOnce_ErrAlreadyRunning(t *testing.T) {
	cfg := newTestConfig(t)
	m := NewManager(cfg)

	if !m.status.TryStart() {
		t.Fatal("expected TryStart to succeed")
	}
	defer m.status.SetSuccess(time.Now().UTC(), "")

	assertLocalizedErrorCode(t, m.RunOnce(context.Background()), "snapshot_already_running")
}

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

func TestManager_RunOnce_PrunesAfterSuccess(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.RetentionCount = 2
	m := NewManager(cfg)

	for i := 0; i < 3; i++ {
		if err := m.RunOnce(context.Background()); err != nil {
			t.Fatalf("RunOnce #%d failed: %v", i, err)
		}
	}

	remaining, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected retention to keep 2 snapshots, got %d", len(remaining))
	}
}

func TestManager_Delete_InvalidID(t *testing.T) {
	backupsDir := t.TempDir()
	m := NewManager(Config{BackupsDir: backupsDir})

	for _, id := range []string{"../etc/passwd", "not-a-snapshot", "snapshot-../evil"} {
		assertLocalizedErrorCode(t, m.Delete(id), "snapshot_invalid_id")
	}
}

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
