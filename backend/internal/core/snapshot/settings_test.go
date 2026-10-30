package snapshot

import (
	"testing"
)

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
