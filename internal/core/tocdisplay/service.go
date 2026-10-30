// Package tocdisplay owns the "always show table of contents" instance-wide
// setting: whether the page viewer's TOC panel/dropdown should be shown for
// every page, regardless of how many headings it has (see
// PageViewer.tsx's default >3-heading threshold).
//
// Unlike internal/core/publicaccess, this has no env/CLI-flag-pinned variant —
// it's a cosmetic display preference, not a security decision, so it is
// always settings-managed: a single JSON file in the data directory that an
// admin can toggle at runtime with no restart.
package tocdisplay

import (
	"encoding/json"
	"fmt"
	"github.com/NianBroken/LeafDoc/internal/core/defaults"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// fileConfig is the on-disk shape of toc-display.json.
type fileConfig struct {
	AlwaysShow bool   `json:"alwaysShow"`
	Mode       string `json:"mode,omitempty"`
}

// UnmarshalJSON distinguishes a missing mode from an explicitly stored legacy boolean.
func (c *fileConfig) UnmarshalJSON(raw []byte) error {
	var value struct {
		AlwaysShow *bool   `json:"alwaysShow"`
		Mode       *string `json:"mode"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	c.Mode = defaults.TocMode
	c.AlwaysShow = false
	if value.AlwaysShow != nil {
		c.AlwaysShow = *value.AlwaysShow
		c.Mode = "auto"
		if c.AlwaysShow {
			c.Mode = "always"
		}
	}
	if value.Mode != nil {
		c.Mode = *value.Mode
	}
	if c.Mode != "auto" && c.Mode != "always" && c.Mode != "never" {
		return sharederrors.NewLocalizedError("toc_display_invalid_mode", "Invalid TOC mode", "invalid toc mode", nil)
	}
	return nil
}

// Service is the process-wide holder of the current always-show-TOC flag.
// Safe for concurrent use — the underlying settings.Store provides locking.
type Service struct {
	store *settings.Store[fileConfig]
}

// New returns a Service whose flag is read from (and written back to)
// <storageDir>/toc-display.json. The initial value is whatever the file
// currently holds; a missing file means disabled.
func New(storageDir string) (*Service, error) {
	st, err := settings.New(storageDir, "toc-display.json", 0o600, fileConfig{Mode: defaults.TocMode}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read toc-display config: %w", err)
	}
	return &Service{store: st}, nil
}

// AlwaysShow reports whether the TOC should always be shown, regardless of
// heading count.
func (s *Service) AlwaysShow() bool {
	return s.Mode() == "always"
}

// SetAlwaysShow persists a new value and updates the in-memory cache.
func (s *Service) SetAlwaysShow(alwaysShow bool) error {
	if err := s.store.Put(fileConfig{AlwaysShow: alwaysShow}); err != nil {
		return sharederrors.NewLocalizedError(
			"toc_display_update_failed",
			"Failed to update TOC display setting",
			"failed to persist toc-display config",
			err,
		)
	}
	return nil
}

// Reload re-reads toc-display.json into the in-memory cache. Called after a
// restore swaps in a different data dir.
func (s *Service) Reload() error {
	if err := s.store.Reload(); err != nil {
		return fmt.Errorf("failed to reload toc-display config: %w", err)
	}
	return nil
}

// Mode maps legacy configurations to their equivalent global display behavior.
func (s *Service) Mode() string {
	value := s.store.Get()
	if value.Mode != "" {
		return value.Mode
	}
	if value.AlwaysShow {
		return "always"
	}
	return "auto"
}

// SetMode persists the initial display behavior; temporary viewer toggles never call it.
func (s *Service) SetMode(mode string) error {
	if mode != "auto" && mode != "always" && mode != "never" {
		return sharederrors.NewLocalizedError("toc_display_invalid_mode", "Invalid TOC mode", "invalid toc mode", nil)
	}
	return s.store.Put(fileConfig{AlwaysShow: mode == "always", Mode: mode})
}
