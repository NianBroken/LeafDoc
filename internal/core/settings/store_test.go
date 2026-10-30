package settings

import (
	"encoding/json"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "settings") }

type testConfig struct {
	Name     string `json:"name"`
	Injected bool   `json:"-"`
}

func TestMySQLSettingsRoundTripReloadAndNoFiles(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	s, err := New(dir, "config.json", 0600, testConfig{Name: "默认"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Get().Name != "默认" {
		t.Fatal(s.Get())
	}
	if err = s.Put(testConfig{Name: "已保存"}); err != nil {
		t.Fatal(err)
	}
	next, err := New(dir, "config.json", 0600, testConfig{}, nil)
	if err != nil || next.Get().Name != "已保存" {
		t.Fatalf("%+v %v", next, err)
	}
	raw, _ := json.Marshal(testConfig{Name: "外部更新"})
	if err = database.WriteRecord(dir, "settings", "config.json", raw); err != nil {
		t.Fatal(err)
	}
	if err = s.Reload(); err != nil || s.Get().Name != "外部更新" {
		t.Fatal(err, s.Get())
	}
	if _, err = os.Stat(filepath.Join(dir, "config.json")); !os.IsNotExist(err) {
		t.Fatal("settings written to disk")
	}
}
func TestMySQLSettingsOnLoadConcurrentAndInvalidData(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	s, err := New(dir, "config.json", 0600, testConfig{}, func(v *testConfig) { v.Injected = true })
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Put(testConfig{Name: "并发"}); err != nil {
				t.Error(err)
			}
			_ = s.Get()
		}()
	}
	wg.Wait()
	if !s.Get().Injected {
		t.Fatal("runtime injection missing")
	}
	if err = database.WriteRecord(dir, "settings", "config.json", []byte("invalid")); err != nil {
		t.Fatal(err)
	}
	if err = s.Reload(); err == nil {
		t.Fatal("invalid configuration accepted")
	}
}

// Missing fields are persisted from defaults; explicit false and zero remain authoritative.
func TestMissingFieldsMaterializeWithoutReplacingExplicitValues(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	type config struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
		Count   int    `json:"count"`
	}
	if err := database.WriteRecord(dir, "settings", "partial", []byte(`{"enabled":false,"count":0}`)); err != nil {
		t.Fatal(err)
	}
	store, err := New(dir, "partial", 0600, config{Name: "默认", Enabled: true, Count: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Get(); got.Name != "默认" || got.Enabled || got.Count != 0 {
		t.Fatal(got)
	}
	raw, err := database.ReadRecord(dir, "settings", "partial")
	if err != nil {
		t.Fatal(err)
	}
	var saved config
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved != store.Get() {
		t.Fatal(saved)
	}
	if err := database.WriteRecord(dir, "settings", "partial", []byte(`{"name":"恢复"}`)); err != nil {
		t.Fatal(err)
	}
	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	if got := store.Get(); !got.Enabled || got.Count != 10 {
		t.Fatal(got)
	}
}
