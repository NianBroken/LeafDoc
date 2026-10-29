package snapshot

import (
	"encoding/json"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"testing"
	"time"
)

func TestSettingsStorePersistsAndAppliesSettings(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	store, err := NewSettingsStore(dir, 24*time.Hour, 5)
	if err != nil {
		t.Fatal(err)
	}
	var applied Settings
	store.SetApply(func(value Settings) { applied = value })
	want := Settings{Enabled: true, Mode: scheduleModeDaily, Hour: 2, TimeZone: "Asia/Shanghai", RetentionCount: 0}
	if err := store.Put(want); err != nil {
		t.Fatal(err)
	}
	if applied != want {
		t.Fatalf("applied settings = %+v, want %+v", applied, want)
	}
	reloaded, err := NewSettingsStore(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Get(); got != want {
		t.Fatalf("persisted settings = %+v, want %+v", got, want)
	}
	restored := want
	restored.Enabled = false
	restored.RetentionCount = 3
	raw, err := json.Marshal(restored)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.WriteRecord(dir, "settings", "snapshot-settings.json", raw); err != nil {
		t.Fatal(err)
	}

	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	if applied.Enabled || applied.RetentionCount != 3 {
		t.Fatalf("reload did not apply restored settings: %+v", applied)
	}
}

func TestValidateSettingsRejectsInvalidValues(t *testing.T) {
	base := Settings{Enabled: true, Mode: scheduleModeDaily, Hour: 2, TimeZone: "UTC", RetentionCount: 0}
	for name, value := range map[string]Settings{
		"negative retention":  func() Settings { v := base; v.RetentionCount = -1; return v }(),
		"excessive retention": func() Settings { v := base; v.RetentionCount = maxRetentionCount + 1; return v }(),
		"invalid timezone":    func() Settings { v := base; v.TimeZone = "Mars/Olympus"; return v }(),
		"local timezone":      func() Settings { v := base; v.TimeZone = "Local"; return v }(),
		"invalid daily hour":  func() Settings { v := base; v.Hour = 24; return v }(),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateSettings(value); err == nil {
				t.Fatal("ValidateSettings() accepted invalid settings")
			}
		})
	}
}
