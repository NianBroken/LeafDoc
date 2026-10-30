package restore

import (
	"archive/zip"

	"errors"

	"os"
	"path/filepath"
	"testing"
	"time"
)

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
