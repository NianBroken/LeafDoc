// Package preferences stores the single instance-wide set of UI preferences.
package preferences

import (
	"fmt"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
)

// Config is public instance configuration and never contains account data or credentials.
type Config struct {
	Language   string `json:"language"`
	DateFormat string `json:"dateFormat"`
	TimeFormat string `json:"timeFormat"`
	AutoSave   bool   `json:"autoSave"`
}

// Service serializes partial updates so concurrent administrators cannot lose fields.
type Service struct {
	mu    sync.Mutex
	store *settings.Store[Config]
}

func Defaults() Config {
	return Config{Language: "zh", DateFormat: "iso", TimeFormat: "24h", AutoSave: true}
}
func New(dir string) (*Service, error) {
	if err := RemoveLegacyState(dir); err != nil {
		return nil, err
	}
	st, err := settings.New(dir, "preferences.json", 0600, Defaults(), nil)
	if err != nil {
		return nil, err
	}
	return &Service{store: st}, nil
}
func (s *Service) Get() Config   { return s.store.Get() }
func (s *Service) Reload() error { return s.store.Reload() }

// Patch applies explicit fields only; format values never depend on device locale.
func (s *Service) Patch(patch map[string]any) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.Get()
	for k, x := range patch {
		switch k {
		case "language":
			val, ok := x.(string)
			if !ok || !oneOf(val, "zh", "en", "de", "es", "fr") {
				return v, fmt.Errorf("invalid language")
			}
			v.Language = val
		case "dateFormat":
			val, ok := x.(string)
			if !ok || !oneOf(val, "iso", "dmy_dot", "mdy_slash", "dmy_slash") {
				return v, fmt.Errorf("invalid date format")
			}
			v.DateFormat = val
		case "timeFormat":
			val, ok := x.(string)
			if !ok || !oneOf(val, "24h", "12h") {
				return v, fmt.Errorf("invalid time format")
			}
			v.TimeFormat = val
		case "autoSave":
			val, ok := x.(bool)
			if !ok {
				return v, fmt.Errorf("invalid autosave")
			}
			v.AutoSave = val
		default:
			return v, fmt.Errorf("unknown preference")
		}
	}
	return v, s.store.Put(v)
}
func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
