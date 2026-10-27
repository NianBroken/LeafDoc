package preferences

import (
	"sync"
	"testing"
)

// Global preferences persist independently from accounts and serialize partial writes.
func TestGlobalPreferences(t *testing.T) {
	dir := t.TempDir()
	svc, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if svc.Get() != Defaults() {
		t.Fatal("unexpected defaults")
	}
	for _, patch := range []map[string]any{{"dateFormat": "locale"}, {"timeFormat": "locale"}, {"language": "unknown"}, {"autoSave": "true"}, {"unknown": true}} {
		if _, err := svc.Patch(patch); err == nil {
			t.Fatalf("accepted invalid patch: %v", patch)
		}
	}
	var wg sync.WaitGroup
	for _, patch := range []map[string]any{{"language": "fr"}, {"dateFormat": "dmy_dot"}, {"timeFormat": "12h"}, {"autoSave": false}} {
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
	want := Config{Language: "fr", DateFormat: "dmy_dot", TimeFormat: "12h", AutoSave: false}
	if reopened.Get() != want {
		t.Fatalf("got %+v, want %+v", reopened.Get(), want)
	}
}
