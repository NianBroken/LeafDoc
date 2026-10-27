package maintenance

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/perber/wiki/internal/core/auth"
	"github.com/perber/wiki/internal/core/shared/sqliteutil"
	"github.com/perber/wiki/internal/testutil"
)

func TestResetAdminPassword(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// First, create an admin user
	store, err := auth.NewUserStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create user store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})

	userService := auth.NewUserService(store)
	_, err = userService.CreateUser("admin", "admin@example.com", "oldpassword", "admin")
	if err != nil {
		_ = store.Close()
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// Close the store before ResetAdminPassword opens it
	err = store.Close()
	if err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Now test ResetAdminPassword
	adminUser, err := ResetAdminPassword(tempDir, "", "")
	if err != nil {
		t.Fatalf("ResetAdminPassword failed: %v", err)
	}

	if adminUser.Username != "admin" {
		t.Errorf("Expected username 'admin', got: %s", adminUser.Username)
	}

	if adminUser.Password == "" {
		t.Errorf("Expected a new password to be generated, got empty string")
	}

	// Verify we can log in with the new password
	store2, err := auth.NewUserStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create user store: %v", err)
	}
	t.Cleanup(func() {
		if err := store2.Close(); err != nil {
			t.Error(err)
		}
	})
	defer func() {
		if err := store2.Close(); err != nil {
			t.Fatalf("Failed to close user store: %v", err)
		}
	}()

	userService2 := auth.NewUserService(store2)
	_, err = userService2.GetUserByEmailOrUsernameAndPassword("admin", adminUser.Password)
	if err != nil {
		t.Errorf("Failed to login with new password: %v", err)
	}
}

func TestResetAdminPassword_NoAdmin(t *testing.T) {
	// Create a temporary directory for the test (no admin user exists)
	tempDir := t.TempDir()

	// Test ResetAdminPassword when no admin exists
	adminUser, err := ResetAdminPassword(tempDir, "", "")
	if err != nil {
		t.Fatalf("ResetAdminPassword failed: %v", err)
	}

	if adminUser.Username != "admin" {
		t.Errorf("Expected username 'admin', got: %s", adminUser.Username)
	}

	if adminUser.Email != "admin@localhost" {
		t.Errorf("Expected email 'admin@localhost', got: %s", adminUser.Email)
	}

	if adminUser.Password == "" {
		t.Errorf("Expected a new password to be generated, got empty string")
	}

	// Verify we can log in with the new password
	store, err := auth.NewUserStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create user store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Failed to close user store: %v", err)
		}
	}()

	userService := auth.NewUserService(store)
	_, err = userService.GetUserByEmailOrUsernameAndPassword("admin", adminUser.Password)
	if err != nil {
		t.Errorf("Failed to login with new password: %v", err)
	}
}

func TestResetAdminPassword_NoAdmin_UsesGivenUsernameAndEmail(t *testing.T) {
	// Create a temporary directory for the test (no admin user exists)
	tempDir := t.TempDir()

	adminUser, err := ResetAdminPassword(tempDir, "root", "root@example.com")
	if err != nil {
		t.Fatalf("ResetAdminPassword failed: %v", err)
	}

	if adminUser.Username != "root" {
		t.Errorf("Expected username 'root', got: %s", adminUser.Username)
	}

	if adminUser.Email != "root@example.com" {
		t.Errorf("Expected email 'root@example.com', got: %s", adminUser.Email)
	}
}

// A corrupt users.db must stop the reset instead of being replaced by an
// empty one with a freshly created admin (ADR-0013).
func TestResetAdminPassword_CorruptUsersDB_FailsAndCreatesNoAdmin(t *testing.T) {
	dir := t.TempDir()
	files := testutil.WriteCorruptSQLiteDB(t, filepath.Join(dir, "users.db"))

	user, err := ResetAdminPassword(dir, "", "")

	if err == nil {
		t.Fatalf("ResetAdminPassword: expected an error, got user %+v", user)
	}
	var corrupt *sqliteutil.CorruptDatabaseError
	if !errors.As(err, &corrupt) {
		t.Fatalf("ResetAdminPassword error = %v (%T), want it to wrap *sqliteutil.CorruptDatabaseError", err, err)
	}
	testutil.AssertFilesUnchanged(t, files)
}

// Recovery preserves account ownership and invalidates credentials issued before it.
func TestResetAdminPassword_ExplicitSecretPreservesIDAndRevokesSessions(t *testing.T) {
	dir := t.TempDir()
	store, err := auth.NewUserStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	original, err := auth.NewUserService(store).CreateUser("owner", "owner@example.com", "old-password", auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	sessions, err := auth.NewSessionStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessions.CreateSession("before-reset", original.ID, "refresh", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Close(); err != nil {
		t.Fatal(err)
	}
	recovered, err := ResetAdminPassword(dir, "admin", "", "recovery-password")
	if err != nil {
		t.Fatal(err)
	}
	if recovered.ID != original.ID || recovered.Username != "admin" || recovered.Email != original.Email {
		t.Fatalf("account ownership changed: %+v", recovered.ToPublicUser())
	}
	store, err = auth.NewUserStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)
	if _, err := auth.NewUserService(store).GetUserByEmailOrUsernameAndPassword("admin", "recovery-password"); err != nil {
		t.Fatal(err)
	}
	sessions, err = auth.NewSessionStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer testutil.WrapCloseWithErrorCheck(sessions.Close, t)
	if active, err := sessions.IsActive("before-reset", original.ID, "refresh", time.Now()); err != nil || active {
		t.Fatalf("old session remains active: %v %v", active, err)
	}
}

// A conflicting username leaves the existing administrator unchanged.
func TestResetAdminPassword_UsernameConflictPreservesPassword(t *testing.T) {
	dir := t.TempDir()
	store, err := auth.NewUserStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer testutil.WrapCloseWithErrorCheck(store.Close, t)
	service := auth.NewUserService(store)
	if _, err := service.CreateUser("owner", "owner@example.com", "old-password", auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateUser("admin", "viewer@example.com", "viewer-password", auth.RoleViewer); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResetAdminUserPassword("admin", "", "recovery-password"); !errors.Is(err, auth.ErrUserAlreadyExists) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if _, err := service.GetUserByEmailOrUsernameAndPassword("owner", "old-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResetAdminUserPassword("owner", "", "short"); !errors.Is(err, auth.ErrPasswordTooShort) {
		t.Fatalf("expected minimum length validation, got %v", err)
	}
}
