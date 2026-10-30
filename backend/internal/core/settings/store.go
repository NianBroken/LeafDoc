// Package settings persists typed instance configuration in MySQL.
package settings

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// Reloadable refreshes a service's configuration after database restoration.
type Reloadable interface{ Reload() error }

// Store caches one typed MySQL setting; the key is stable across operating systems.
type Store[T any] struct {
	directory, key string
	onLoad         func(*T)
	mu             sync.RWMutex
	val, zero      T
}

// New loads the database record and materializes the supplied defaults when absent.
func New[T any](dir, key string, _ os.FileMode, zero T, onLoad func(*T)) (*Store[T], error) {
	s := &Store[T]{directory: dir, key: key, onLoad: onLoad, val: zero, zero: zero}
	if err := s.reloadLocked(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Store[T]) Get() T { s.mu.RLock(); defer s.mu.RUnlock(); return s.val }
func (s *Store[T]) Put(v T) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err = database.WriteRecord(s.directory, "settings", s.key, raw); err != nil {
		return err
	}
	s.val = v
	if s.onLoad != nil {
		s.onLoad(&s.val)
	}
	notifyCommitted()
	return nil
}
func (s *Store[T]) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reloadLocked(); err != nil {
		return err
	}
	notifyCommitted()
	return nil
}
func (s *Store[T]) reloadLocked() error {
	raw, err := database.ReadRecord(s.directory, "settings", s.key)
	if errors.Is(err, sql.ErrNoRows) {
		s.val = s.zero
		raw, err = json.Marshal(s.zero)
		if err != nil {
			return err
		}
		if err = database.WriteRecord(s.directory, "settings", s.key, raw); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("read setting %s: %w", s.key, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("invalid setting %s: %w", s.key, err)
	}
	if fields == nil {
		return fmt.Errorf("setting %s must be an object", s.key)
	}
	for field, value := range fields {
		if string(value) == "null" {
			return fmt.Errorf("setting %s field %s must not be null", s.key, field)
		}
	}
	s.val = s.zero
	if err = json.Unmarshal(raw, &s.val); err != nil {
		return err
	}
	if s.onLoad != nil {
		s.onLoad(&s.val)
	}
	// Persist materialized fields while preserving explicit false and zero values.
	completed, err := json.Marshal(s.val)
	if err != nil {
		return err
	}
	if string(completed) != string(raw) {
		if err := database.WriteRecord(s.directory, "settings", s.key, completed); err != nil {
			return err
		}
	}
	return nil
}
