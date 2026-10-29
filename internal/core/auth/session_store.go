package auth

import (
	"context"
	"database/sql"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"log/slog"
	"sync"
	"time"
)

type SessionStore struct {
	mu         sync.Mutex
	storageDir string
	db         *sql.DB
	cancel     context.CancelFunc
	done       chan struct{}
	log        *slog.Logger
}

func NewSessionStore(storageDir string) (*SessionStore, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &SessionStore{
		storageDir: storageDir,
		cancel:     cancel,
		done:       make(chan struct{}),
		log:        slog.Default().With("component", "SessionStore"),
	}

	err := s.ensureSchema()
	if err != nil {
		cancel()
		return nil, err
	}

	// Cleanup expired sessions periodically
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.CleanupExpiredSessions(); err != nil {
					s.log.Warn("failed to cleanup expired sessions", "error", err)
				}
			}
		}
	}()

	return s, nil

}

func (s *SessionStore) withDB(fn func(db *sql.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		db, err := database.Open(s.storageDir)
		if err != nil {
			return err
		}
		s.db = db
	}

	return fn(s.db)
}

func (s *SessionStore) ensureSchema() error {
	return s.withDB(func(db *sql.DB) error { return db.Ping() })
}

func (s *SessionStore) Close() error {
	// Signal the cleanup goroutine to stop
	s.cancel()
	// Wait for the cleanup goroutine to finish
	<-s.done
	// Close the database connection
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

func (s *SessionStore) CreateSession(id, userID, tokenType string, expiresAt time.Time) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO sessions (id, user_id, token_type, created_at, expires_at, revoked_at)
			VALUES (?, ?, ?, ?, ?, NULL);
		`, id, userID, tokenType, time.Now().Unix(), expiresAt.Unix())
		return err
	})
}

func (s *SessionStore) IsActive(id, userID, tokenType string, now time.Time) (bool, error) {
	var expiresAt int64
	var revokedAt sql.NullInt64

	err := s.withDB(func(db *sql.DB) error {
		return db.QueryRow(`
			SELECT expires_at, revoked_at
			FROM sessions
			WHERE id = ? AND user_id = ? AND token_type = ?;
		`, id, userID, tokenType).Scan(&expiresAt, &revokedAt)
	})

	if err == sql.ErrNoRows {
		// no such session
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if revokedAt.Valid {
		return false, nil
	}
	if now.Unix() > expiresAt {
		return false, nil
	}
	return true, nil
}

func (s *SessionStore) RevokeSession(id string) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			UPDATE sessions
			SET revoked_at = ?
			WHERE id = ? AND revoked_at IS NULL;
		`, time.Now().Unix(), id)
		return err
	})
}

func (s *SessionStore) RevokeAllSessionsForUser(userID string) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			UPDATE sessions
			SET revoked_at = ?
			WHERE user_id = ? AND revoked_at IS NULL;
		`, time.Now().Unix(), userID)
		return err
	})
}

// RevokeAllSessionsForUserExcept revokes every active session for userID
// except the one identified by exceptID. If exceptID is empty, every session
// is revoked (same as RevokeAllSessionsForUser) — the safe fallback when the
// caller could not identify which session to preserve.
func (s *SessionStore) RevokeAllSessionsForUserExcept(userID, exceptID string) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			UPDATE sessions
			SET revoked_at = ?
			WHERE user_id = ? AND revoked_at IS NULL AND id != ?;
		`, time.Now().Unix(), userID, exceptID)
		return err
	})
}

// DeleteAllSessions invalidates every session after a database restore.
func (s *SessionStore) DeleteAllSessions() error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`DELETE FROM sessions;`)
		return err
	})
}

func (s *SessionStore) CleanupExpiredSessions() error {
	now := time.Now()
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			DELETE FROM sessions
			WHERE expires_at <= ?;
		`, now.Unix())
		return err
	})
}

// Generation reads the persisted authentication epoch. An empty fresh database
// receives one random UUID; reopening a store never rotates an existing epoch.
func (s *SessionStore) Generation() (string, error) {
	var generation string
	err := s.withDB(func(db *sql.DB) error {
		err := db.QueryRow("SELECT generation FROM auth_state WHERE id=1").Scan(&generation)
		if err != sql.ErrNoRows {
			return err
		}
		if _, err := db.Exec("INSERT INTO auth_state(id,generation) VALUES(1,UUID()) ON DUPLICATE KEY UPDATE generation=auth_state.generation"); err != nil {
			return err
		}
		return db.QueryRow("SELECT generation FROM auth_state WHERE id=1").Scan(&generation)
	})
	return generation, err
}
