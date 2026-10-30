package preferences

import (
	"encoding/json"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"sync"
	"testing"
)

// Loading and restore reloads persist only the current preference schema.
func TestPreferencesNormalizeStoredFields(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	legacy := []byte(`{"language":"fr","dateFormat":"iso","timeFormat":"24h","autoSave":false}`)
	if err := database.WriteRecord(dir, "settings", "preferences.json", legacy); err != nil {
		t.Fatal(err)
	}
	svc, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := database.ReadRecord(dir, "settings", "preferences.json")
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if len(fields) != 3 || fields["language"] != "fr" {
			t.Fatalf("unexpected persisted fields: %s", raw)
		}
		if err := database.WriteRecord(dir, "settings", "preferences.json", legacy); err != nil {
			t.Fatal(err)
		}
		if err := svc.Reload(); err != nil {
			t.Fatal(err)
		}
	}
}

// Global preferences persist independently from accounts and serialize partial writes.
func TestGlobalPreferences(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	svc, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if svc.Get() != Defaults() {
		t.Fatal("unexpected defaults")
	}
	for _, patch := range []map[string]any{{"dateFormat": "locale"}, {"timeFormat": "locale"}, {"language": "unknown"}, {"removedSetting": true}, {"unknown": true}} {
		if _, err := svc.Patch(patch); err == nil {
			t.Fatalf("accepted invalid patch: %v", patch)
		}
	}
	var wg sync.WaitGroup
	for _, patch := range []map[string]any{{"language": "fr"}, {"dateFormat": "dmy_dot"}, {"timeFormat": "12h"}} {
		wg.Add(1)
		go func(p map[string]any) {
			defer wg.Done()
			if _, err := svc.Patch(p); err != nil {
				t.Error(err)
			}
		}(patch)
	}
	wg.Wait()
	reopened, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Language: "fr", DateFormat: "dmy_dot", TimeFormat: "12h"}
	if reopened.Get() != want {
		t.Fatalf("got %+v, want %+v", reopened.Get(), want)
	}
}
