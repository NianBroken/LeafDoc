package snapshot

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/perber/wiki/internal/core/settings"
)

const (
	scheduleModeInterval = "interval"
	scheduleModeDaily    = "daily"
	maxRetentionCount    = 100000
	minSnapshotInterval  = time.Minute
)

type Settings struct {
	Enabled        bool   `json:"enabled"`
	RunOnStartup   bool   `json:"runOnStartup"`
	Mode           string `json:"mode"`
	Interval       string `json:"interval"`
	Hour           int    `json:"hour"`
	Minute         int    `json:"minute"`
	TimeZone       string `json:"timeZone"`
	RetentionCount int    `json:"retentionCount"`
}

type SettingsStore struct {
	store *settings.Store[Settings]
	mu    sync.RWMutex
	apply func(Settings)
}

func NewSettingsStore(dataDir string, interval time.Duration, retentionCount int) (*SettingsStore, error) {
	if interval < 0 {
		interval = 0
	} else if interval > 0 && interval < minSnapshotInterval {
		interval = minSnapshotInterval
	}
	if retentionCount < 0 {
		retentionCount = 0
	}
	defaults := Settings{
		Enabled:        interval > 0,
		RunOnStartup:   true,
		Mode:           scheduleModeInterval,
		Interval:       interval.String(),
		Hour:           2,
		TimeZone:       "UTC",
		RetentionCount: retentionCount,
	}
	store, err := settings.New(dataDir, "snapshot-settings.json", 0o600, defaults, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot settings: %w", err)
	}
	value := store.Get()
	if err := ValidateSettings(value); err != nil {
		return nil, fmt.Errorf("invalid snapshot settings: %w", err)
	}
	return &SettingsStore{store: store}, nil
}

func (s *SettingsStore) Get() Settings {
	return s.store.Get()
}

func (s *SettingsStore) Put(value Settings) error {
	if err := ValidateSettings(value); err != nil {
		return err
	}
	if err := s.store.Put(value); err != nil {
		return err
	}
	s.applySettings(value)
	return nil
}

func (s *SettingsStore) Reload() error {
	if err := s.store.Reload(); err != nil {
		return err
	}
	s.applySettings(s.store.Get())
	return nil
}

func (s *SettingsStore) SetApply(apply func(Settings)) {
	s.mu.Lock()
	s.apply = apply
	s.mu.Unlock()
	s.applySettings(s.store.Get())
}

func (s *SettingsStore) applySettings(value Settings) {
	s.mu.RLock()
	apply := s.apply
	s.mu.RUnlock()
	if apply != nil {
		apply(value)
	}
}

func ValidateSettings(value Settings) error {
	if value.Mode != scheduleModeInterval && value.Mode != scheduleModeDaily {
		return fmt.Errorf("schedule mode must be interval or daily")
	}
	if value.RetentionCount < 0 || value.RetentionCount > maxRetentionCount {
		return fmt.Errorf("retention count must be between 0 and %d", maxRetentionCount)
	}
	if value.Enabled && value.Mode == scheduleModeInterval {
		interval, err := time.ParseDuration(strings.TrimSpace(value.Interval))
		if err != nil || interval < minSnapshotInterval {
			return fmt.Errorf("interval must be at least %s", minSnapshotInterval)
		}
	}
	if value.Mode == scheduleModeDaily {
		if value.Hour < 0 || value.Hour > 23 || value.Minute < 0 || value.Minute > 59 {
			return fmt.Errorf("daily schedule time is invalid")
		}
		if strings.TrimSpace(value.TimeZone) == "" || strings.TrimSpace(value.TimeZone) == "Local" {
			return fmt.Errorf("time zone is required")
		}
		if _, err := time.LoadLocation(value.TimeZone); err != nil {
			return fmt.Errorf("invalid IANA time zone: %w", err)
		}
	}
	return nil
}
