// Package preferences stores the single instance-wide set of UI preferences.
package preferences

import (
	"github.com/NianBroken/LeafDoc/internal/core/defaults"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
)

// Config is public instance configuration and never contains account data or credentials.
type Config struct {
	Language   string `json:"language"`
	DateFormat string `json:"dateFormat"`
	TimeFormat string `json:"timeFormat"`
}

// Service serializes partial updates so concurrent administrators cannot lose fields.
type Service struct {
	mu    sync.Mutex
	store *settings.Store[Config]
}

func Defaults() Config {
	return Config{Language: defaults.Language, DateFormat: defaults.DateFormat, TimeFormat: defaults.TimeFormat}
}
func New(dir string) (*Service, error) {
	st, err := settings.New(dir, "preferences.json", 0600, Defaults(), nil)
	if err != nil {
		return nil, err
	}
	if err := validate(st.Get()); err != nil {
		return nil, err
	}
	if err := st.Put(st.Get()); err != nil {
		return nil, err
	}
	return &Service{store: st}, nil
}
func (s *Service) Get() Config { return s.store.Get() }
func (s *Service) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Reload(); err != nil {
		return err
	}
	if err := validate(s.store.Get()); err != nil {
		return err
	}
	return s.store.Put(s.store.Get())
}

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
				return v, invalidChoice("invalid language")
			}
			v.Language = val
		case "dateFormat":
			val, ok := x.(string)
			if !ok || !oneOf(val, "iso", "dmy_dot", "mdy_slash", "dmy_slash") {
				return v, invalidChoice("invalid date format")
			}
			v.DateFormat = val
		case "timeFormat":
			val, ok := x.(string)
			if !ok || !oneOf(val, "24h", "12h") {
				return v, invalidChoice("invalid time format")
			}
			v.TimeFormat = val
		default:
			return v, invalidChoice("unknown preference")
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

// validate rejects persisted invalid choices instead of silently changing them.
func validate(value Config) error {
	if !oneOf(value.Language, "zh", "en", "de", "es", "fr") {
		return invalidChoice("invalid language")
	}
	if !oneOf(value.DateFormat, "iso", "dmy_dot", "mdy_slash", "dmy_slash") {
		return invalidChoice("invalid date format")
	}
	if !oneOf(value.TimeFormat, "24h", "12h") {
		return invalidChoice("invalid time format")
	}
	return nil
}

// Invalid preference choices have a stable client-facing cause; storage errors remain server errors.
func invalidChoice(message string) error {
	return sharederrors.NewLocalizedError("invalid_preferences", message, message, nil)
}
