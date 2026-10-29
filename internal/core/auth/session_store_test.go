package auth

import (
	"os"
	"testing"
	"time"

	"github.com/NianBroken/LeafDoc/internal/testutil"
)

func TestSessionStore_UsesInnoDB(t *testing.T) {
	testutil.ResetMySQL(t)
	store, err := NewSessionStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var engine string
	if err := store.db.QueryRow("SELECT ENGINE FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", "sessions").Scan(&engine); err != nil {
		t.Fatal(err)
	}
	if engine != "InnoDB" {
		t.Fatalf("engine=%s", engine)
	}
}

func TestSessionStore_CreateAndValidateSession(t *testing.T) {
	testutil.ResetMySQL(t)
	store, err := NewSessionStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewSessionStore err: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)

	expiresAt := time.Now().Add(time.Hour)
	if err := store.CreateSession("s1", "u1", "refresh", expiresAt); err != nil {
		t.Fatalf("CreateSession err: %v", err)
	}

	active, err := store.IsActive("s1", "u1", "refresh", time.Now())
	if err != nil {
		t.Fatalf("IsActive err: %v", err)
	}
	if !active {
		t.Fatalf("expected session to be active")
	}
}

func TestSessionStore_RevokeAllSessionsForUserExcept(t *testing.T) {
	testutil.ResetMySQL(t)
	store, err := NewSessionStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewSessionStore err: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)

	expiresAt := time.Now().Add(time.Hour)
	if err := store.CreateSession("keep", "u1", "refresh", expiresAt); err != nil {
		t.Fatalf("CreateSession(keep) err: %v", err)
	}
	if err := store.CreateSession("revoke1", "u1", "refresh", expiresAt); err != nil {
		t.Fatalf("CreateSession(revoke1) err: %v", err)
	}
	if err := store.CreateSession("revoke2", "u1", "login_challenge", expiresAt); err != nil {
		t.Fatalf("CreateSession(revoke2) err: %v", err)
	}
	// A different user's session must never be touched by userID-scoped revocation.
	if err := store.CreateSession("other-user", "u2", "refresh", expiresAt); err != nil {
		t.Fatalf("CreateSession(other-user) err: %v", err)
	}

	if err := store.RevokeAllSessionsForUserExcept("u1", "keep"); err != nil {
		t.Fatalf("RevokeAllSessionsForUserExcept err: %v", err)
	}

	active, err := store.IsActive("keep", "u1", "refresh", time.Now())
	if err != nil || !active {
		t.Fatalf("expected excepted session to remain active, active=%v err=%v", active, err)
	}
	if active, _ := store.IsActive("revoke1", "u1", "refresh", time.Now()); active {
		t.Fatal("expected revoke1 to be revoked")
	}
	if active, _ := store.IsActive("revoke2", "u1", "login_challenge", time.Now()); active {
		t.Fatal("expected revoke2 to be revoked")
	}
	if active, _ := store.IsActive("other-user", "u2", "refresh", time.Now()); !active {
		t.Fatal("expected other user's session to be unaffected")
	}
}

func TestSessionStore_RevokeAllSessionsForUserExcept_EmptyExceptRevokesAll(t *testing.T) {
	testutil.ResetMySQL(t)
	store, err := NewSessionStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewSessionStore err: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)

	expiresAt := time.Now().Add(time.Hour)
	if err := store.CreateSession("s1", "u1", "refresh", expiresAt); err != nil {
		t.Fatalf("CreateSession err: %v", err)
	}

	if err := store.RevokeAllSessionsForUserExcept("u1", ""); err != nil {
		t.Fatalf("RevokeAllSessionsForUserExcept err: %v", err)
	}
	if active, _ := store.IsActive("s1", "u1", "refresh", time.Now()); active {
		t.Fatal("expected all sessions revoked when exceptID is empty")
	}
}

func TestSessionStore_CreatesNoLocalDatabase(t *testing.T) {
	testutil.ResetMySQL(t)
	dir := t.TempDir()
	store, err := NewSessionStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected local persistence: %v", entries)
	}
}
