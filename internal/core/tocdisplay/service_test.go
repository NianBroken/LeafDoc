package tocdisplay

import (
	"encoding/json"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestService_NoFile_DisabledByDefault(t *testing.T) {
	testutil.ResetMySQL(t)
	svc, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if svc.AlwaysShow() {
		t.Fatal("expected AlwaysShow() false when toc-display.json is absent")
	}
}

func TestService_SetAlwaysShow_PersistsAcrossReconstruction(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()

	svc, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := svc.SetAlwaysShow(true); err != nil {
		t.Fatalf("SetAlwaysShow(true): %v", err)
	}
	if !svc.AlwaysShow() {
		t.Fatal("expected AlwaysShow() true right after SetAlwaysShow(true)")
	}

	reopened, err := New(dir)
	if err != nil {
		t.Fatalf("New (reopen): %v", err)
	}
	if !reopened.AlwaysShow() {
		t.Fatal("expected persisted AlwaysShow() true after reconstruction")
	}

	if err := reopened.SetAlwaysShow(false); err != nil {
		t.Fatalf("SetAlwaysShow(false): %v", err)
	}
	again, err := New(dir)
	if err != nil {
		t.Fatalf("New (reopen 2): %v", err)
	}
	if again.AlwaysShow() {
		t.Fatal("expected persisted AlwaysShow() false after SetAlwaysShow(false)")
	}
}

func TestService_SetAlwaysShow_PersistsMySQLWithoutLocalConfig(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	svc, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetAlwaysShow(true); err != nil {
		t.Fatal(err)
	}
	data, err := database.ReadRecord(dir, "settings", "toc-display.json")
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]bool
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	if !value["alwaysShow"] {
		t.Fatalf("unexpected persisted configuration: %s", data)
	}
	if _, err := os.Stat(filepath.Join(dir, "toc-display.json")); !os.IsNotExist(err) {
		t.Fatalf("local config exists: %v", err)
	}
}

func TestService_Reload_PicksUpExternalFileChange(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	svc, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if svc.AlwaysShow() {
		t.Fatal("precondition: expected disabled")
	}

	// Simulate a restore dropping in a different toc-display.json.
	if err := database.WriteRecord(dir, "settings", "toc-display.json", []byte(`{"alwaysShow": true}`)); err != nil {
		t.Fatalf("write external file: %v", err)
	}
	if svc.AlwaysShow() {
		t.Fatal("expected cache to still read false before Reload()")
	}

	if err := svc.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if !svc.AlwaysShow() {
		t.Fatal("expected AlwaysShow() true after Reload() picked up the new file")
	}
}

func TestService_Load_CorruptFileIsAnError(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	if err := database.WriteRecord(dir, "settings", "toc-display.json", []byte("not json")); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}
	if _, err := New(dir); err == nil {
		t.Fatal("expected New to fail on an unparseable toc-display.json")
	}
}

func TestService_ConcurrentAccess_IsRaceFree(t *testing.T) {
	testutil.ResetMySQL(t)
	svc, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(v int) { defer wg.Done(); _ = svc.SetAlwaysShow(v%2 == 0) }(i)
		go func() { defer wg.Done(); _ = svc.AlwaysShow() }()
	}
	wg.Wait()
}
