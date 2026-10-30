// Package favorites stores each user's private set of favorited pages.
// Unlike tags/links/properties/search, this data is not derived from the
// filesystem tree and must never be touched by resync (see ADR-0001).
package favorites

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

const logCloseRowsFailed = "could not close rows"

// ErrCodeFavoritesStoreUnavailable identifies errFavoritesStoreUnavailable's
// LocalizedError. Mirrors ErrCodeAPIKeyStoreUnavailable.
const ErrCodeFavoritesStoreUnavailable = "favorites_store_unavailable"

// errFavoritesStoreUnavailable is returned while the store is suspended for
// an in-progress live restore (see FavoritesStore.PauseForSwap). Mirrors
// errAPIKeyStoreUnavailable.
func errFavoritesStoreUnavailable() error {
	return sharederrors.NewLocalizedError(
		ErrCodeFavoritesStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"favorites store is suspended for an in-progress restore",
		nil,
	)
}

type FavoritesStore struct {
	executor database.Executor
	mu       sync.Mutex
	db       *sql.DB
	// suspended is set by PauseForSwap and makes withDB refuse to serve
	// queries against a possibly-stale db handle until Replace reopens it.
	suspended bool
	log       *slog.Logger
}

// NewFavoritesStore connects the domain store to the configured MySQL database.
func NewFavoritesStore(storageDir string, log *slog.Logger) (*FavoritesStore, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &FavoritesStore{db: db, log: log}, nil
}

// withDB runs fn against the current db handle, or returns
// errFavoritesStoreUnavailable if the store is suspended for an in-progress
// restore. Mirrors APIKeyStore.withDB.
func (s *FavoritesStore) withDB(fn func(db *sql.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.suspended || s.db == nil {
		return errFavoritesStoreUnavailable()
	}
	return fn(s.db)
}

// Add favorites pageID for userID. Idempotent — favoriting an already-favorited page is a no-op.
func (s *FavoritesStore) Add(userID, pageID string) error {
	err := s.withExecutor(func(db database.Executor) error {
		_, err := db.Exec(
			`INSERT INTO favorites (user_id, page_id, created_at) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE created_at=favorites.created_at`,
			userID, pageID, time.Now().UTC(),
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to add favorite for user %s, page %s: %w", userID, pageID, err)
	}
	return nil
}

// Remove un-favorites pageID for userID. Idempotent — removing a non-favorited page is a no-op.
func (s *FavoritesStore) Remove(userID, pageID string) error {
	err := s.withExecutor(func(db database.Executor) error {
		_, err := db.Exec(`DELETE FROM favorites WHERE user_id = ? AND page_id = ?`, userID, pageID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to remove favorite for user %s, page %s: %w", userID, pageID, err)
	}
	return nil
}

// ListPageIDsForUser returns the page IDs favorited by userID, most recently favorited first.
func (s *FavoritesStore) ListPageIDsForUser(userID string) ([]string, error) {
	var pageIDs []string
	err := s.withExecutor(func(db database.Executor) error {
		rows, err := db.Query(
			`SELECT page_id FROM favorites WHERE user_id = ? ORDER BY created_at DESC`,
			userID,
		)
		if err != nil {
			return err
		}
		defer shared.LogClose(rows.Close, logCloseRowsFailed)

		for rows.Next() {
			var pageID string
			if err := rows.Scan(&pageID); err != nil {
				return err
			}
			pageIDs = append(pageIDs, pageID)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return pageIDs, nil
}

// DeleteAllForPage removes every user's favorite of pageID. Called on page delete.
func (s *FavoritesStore) DeleteAllForPage(pageID string) error {
	err := s.withExecutor(func(db database.Executor) error {
		_, err := db.Exec(`DELETE FROM favorites WHERE page_id = ?`, pageID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete favorites for page %s: %w", pageID, err)
	}
	return nil
}

// DeleteAllForUser removes every favorite belonging to userID. Called on user delete.
func (s *FavoritesStore) DeleteAllForUser(userID string) error {
	err := s.withExecutor(func(db database.Executor) error {
		_, err := db.Exec(`DELETE FROM favorites WHERE user_id = ?`, userID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete favorites for user %s: %w", userID, err)
	}
	return nil
}

// PauseForSwap prevents access while restored records are committed.
func (s *FavoritesStore) PauseForSwap() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.suspended = true
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// Replace opens a fresh MySQL handle and resumes favorite access.
func (s *FavoritesStore) Replace(storageDir string) error {
	fresh, err := NewFavoritesStore(storageDir, s.log)
	if err != nil {
		return err
	}

	s.mu.Lock()
	old := s.db
	s.db = fresh.db
	s.suspended = false
	s.mu.Unlock()

	if old != nil {
		if err := old.Close(); err != nil && s.log != nil {
			s.log.Warn("failed to close previous favorites store after restore", "error", err)
		}
	}
	return nil
}

func (s *FavoritesStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return err
		}
		s.db = nil
	}
	return nil
}

// WithExecutor creates a transaction-scoped favorite store without owning its connection.
func (s *FavoritesStore) WithExecutor(executor database.Executor) *FavoritesStore {
	return &FavoritesStore{executor: executor, log: s.log}
}

func (s *FavoritesStore) withExecutor(fn func(database.Executor) error) error {
	if s.executor != nil {
		return fn(s.executor)
	}
	return s.withDB(func(db *sql.DB) error { return fn(db) })
}
