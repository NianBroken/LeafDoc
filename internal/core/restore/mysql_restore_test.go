package restore

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/shared"
	"github.com/NianBroken/LeafDoc/internal/core/snapshot"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "restore") }

// backupFixture captures real records and resources before later mutation.
func backupFixture(t *testing.T) (string, *snapshot.Manager, string) {
	t.Helper()
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	assets := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assets, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "example.txt"), []byte("original resource"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := database.WriteRecord(dir, "pages", "one", []byte("原始正文")); err != nil {
		t.Fatal(err)
	}
	manager := snapshot.NewManager(snapshot.Config{DataDir: dir, BackupsDir: filepath.Join(dir, "snapshots"), AssetsDir: assets, WikiVersion: "test"})
	if err := manager.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries, err := manager.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("snapshot %v %v", entries, err)
	}
	return dir, manager, entries[0].ID
}

func TestOfflineRestoreMySQLAndResources(t *testing.T) {
	dir, manager, id := backupFixture(t)
	if err := database.WriteRecord(dir, "pages", "one", []byte("changed")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "later.txt"), []byte("later resource"), 0644); err != nil {
		t.Fatal(err)
	}
	path, err := manager.SnapshotZipPath(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := RestoreOffline(dir, path); err != nil {
		t.Fatal(err)
	}
	raw, err := database.ReadRecord(dir, "pages", "one")
	if err != nil || string(raw) != "原始正文" {
		t.Fatalf("restore %s %v", raw, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "assets", "later.txt")); !os.IsNotExist(err) {
		t.Fatalf("resource not in backup retained: %v", err)
	}
}

func TestLiveRestoreAndUpload(t *testing.T) {
	for _, upload := range []bool{false, true} {
		t.Run(map[bool]string{false: "id", true: "upload"}[upload], func(t *testing.T) {
			dir, snapshots, id := backupFixture(t)
			path, err := snapshots.SnapshotZipPath(id)
			if err != nil {
				t.Fatal(err)
			}
			manager := NewManager(Config{DataDir: dir, SnapshotManager: snapshots, WriteGate: NewWriteGate()})
			if upload {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				err = manager.TriggerRestoreFromUpload(bytes.NewReader(raw))
			} else {
				err = manager.TriggerRestore(id)
			}
			if err != nil {
				t.Fatal(err)
			}
			manager.Wait()
			if state := manager.Status(); state.Error != "" || !state.Done {
				t.Fatalf("restore status %+v", state)
			}
			if manager.cfg.WriteGate.Engaged() {
				t.Fatal("write barrier not released")
			}
		})
	}
}

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
				target.Write([]byte("{}"))
			}
			writer.Close()
			file.Close()
			if staging, _, err := extractAndValidate(file.Name(), dir); err == nil {
				os.RemoveAll(staging)
				t.Fatal("unsafe or invalid archive accepted")
			}
		})
	}
}

func TestChecksumTamperingLeavesLiveData(t *testing.T) {
	dir, manager, id := backupFixture(t)
	path, err := manager.SnapshotZipPath(id)
	if err != nil {
		t.Fatal(err)
	}
	staging, _, err := extractAndValidate(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(staging)
	if err := os.WriteFile(filepath.Join(staging, "database.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateArchive(staging); err == nil {
		t.Fatal("tampered data passed validation")
	}
	raw, err := database.ReadRecord(dir, "pages", "one")
	if err != nil || string(raw) != "原始正文" {
		t.Fatal("validation changed live data")
	}
}

func TestExtractionBudget(t *testing.T) {
	dir, manager, id := backupFixture(t)
	path, err := manager.SnapshotZipPath(id)
	if err != nil {
		t.Fatal(err)
	}
	limits := shared.DefaultZipExtractionLimits
	limits.MaxTotalBytes = 1
	if staging, _, err := extractAndValidateWithLimits(path, dir, limits); err == nil {
		os.RemoveAll(staging)
		t.Fatal("extraction budget ignored")
	}
}

func TestResourceRollback(t *testing.T) {
	dir := t.TempDir()
	staging := t.TempDir()
	for _, base := range []string{dir, staging} {
		if err := os.MkdirAll(filepath.Join(base, "assets"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	original := filepath.Join(dir, "assets", "original.txt")
	os.WriteFile(original, []byte("original"), 0644)
	swap := newSwapper(dir, staging)
	calls := 0
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

func TestRestoreRejectsConcurrentJob(t *testing.T) {
	manager := NewManager(Config{})
	manager.job.Start()
	if !errors.Is(manager.TriggerRestore("anything"), ErrRestoreAlreadyRunning) {
		t.Fatal("concurrent restore accepted")
	}
}

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
