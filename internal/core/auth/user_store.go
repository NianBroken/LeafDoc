package auth

import (
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/shared/sqliteutil"
	_ "modernc.org/sqlite"
)

const logCloseRowsFailed = "could not close rows"

type UserStore struct {
	mu         sync.Mutex
	storageDir string
	filename   string
	db         *sql.DB
	// suspended is set by suspend() and makes Connect() refuse to lazily
	// reopen db — see suspend's doc comment.
	suspended bool
}

func databasePath(storageDir string, filename string) string {
	normalizedStorageDir := filepath.FromSlash(strings.ReplaceAll(storageDir, `\`, `/`))
	return filepath.Join(normalizedStorageDir, filename)
}

func NewUserStore(storageDir string) (*UserStore, error) {
	u := &UserStore{
		storageDir: storageDir,
		filename:   "users.db",
	}

	// ensureSchema calls Connect() itself (idempotently), so it alone is
	// enough to bring the store to a working state.
	err := sqliteutil.OpenWithPolicy(databasePath(u.storageDir, u.filename), sqliteutil.FailOnCorruption, func() error {
		if err := u.ensureSchema(); err != nil {
			if u.db != nil {
				_ = u.db.Close()
				u.db = nil
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (f *UserStore) Connect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.suspended {
		return errUserStoreUnavailable()
	}
	if f.db != nil {
		return nil
	}
	db, err := sql.Open("sqlite", databasePath(f.storageDir, f.filename)+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return err
	}
	f.db = db
	return nil
}

// suspend closes db (if open) and marks the store so Connect() refuses to
// lazily reopen it — unlike a plain Close(), which reconnects on the next
// query.
// Used by AuthService.PauseUserStoreForSwap before a live restore renames
// users.db out from under this store: on Windows, an open file handle (or
// one silently reopened by a query landing mid-swap) blocks the rename with
// a sharing violation, which POSIX doesn't have. suspend is permanent for
// this *UserStore instance — restore always continues with a brand new one
// afterward (see AuthService.ReplaceUserStore), so there's no un-suspend.
func (f *UserStore) suspend() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.suspended = true
	if f.db == nil {
		return nil
	}
	err := f.db.Close()
	f.db = nil
	return err
}

func (f *UserStore) ensureSchema() error {
	err := f.Connect()
	if err != nil {
		return err
	}
	_, err = f.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			must_set_password INTEGER NOT NULL DEFAULT 0,
            must_change_password INTEGER NOT NULL DEFAULT 0
		);
	`)
	if err != nil {
		return err
	}
	if err := f.ensureCredentialSchema(); err != nil {
		return err
	}
	return f.ensureMustSetPasswordColumn()
}
func (f *UserStore) ensureMustSetPasswordColumn() error {
	existing, err := f.existingColumns()
	if err != nil {
		return err
	}
	if existing["must_set_password"] {
		return nil
	}
	if _, err := f.db.Exec(`ALTER TABLE users ADD COLUMN must_set_password INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("failed to add column must_set_password to users table: %w", err)
	}
	return nil
}

func (f *UserStore) existingColumns() (map[string]bool, error) {
	rows, err := f.db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	cols := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

func (f *UserStore) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.db != nil {
		err := f.db.Close()
		if err != nil {
			return err
		}
		f.db = nil
	}
	return nil
}

func (f *UserStore) CreateUser(user *User) error {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return err
	}
	// Insert the user into the database
	_, err = f.db.Exec(`
		INSERT INTO users (id, username, password, email, role, must_change_password)
		VALUES (?, ?, ?, ?, ?, ?);
	`, user.ID, user.Username, user.Password, user.Email, user.Role, user.MustChangePassword)
	if err != nil {
		return f.mapConstraintViolationToError(err)
	}
	return nil
}

const userColumns = `id, username, password, email, role, must_set_password, must_change_password`

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanUser scans a row produced by a query selecting userColumns into a User.
func scanUser(row scanner) (*User, error) {
	user := &User{}
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role, &user.MustSetPassword, &user.MustChangePassword)
	return user, err
}

func (f *UserStore) GetUserByID(id string) (*User, error) {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return nil, err
	}

	// Query the user by ID
	row := f.db.QueryRow(`SELECT `+userColumns+`
		FROM users
		WHERE id = ?;
	`, id)

	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (f *UserStore) GetUserByUsername(username string) (*User, error) {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return nil, err
	}
	// Query the user by username
	row := f.db.QueryRow(`SELECT `+userColumns+`
		FROM users
		WHERE username = ?;
	`, username)

	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (f *UserStore) GetUserByEmail(email string) (*User, error) {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return nil, err
	}
	// Query the user by email
	row := f.db.QueryRow(`SELECT `+userColumns+`
		FROM users
		WHERE email = ?;
	`, email)

	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (f *UserStore) UpdateUser(user *User) error {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return err
	}

	// Check if a user with the given ID exists
	existingUser, err := f.GetUserByID(user.ID)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// Update the user in the database
	result, err := f.db.Exec(`
		UPDATE users
		SET username = ?, password = ?, email = ?, role = ?
		WHERE id = ?
		  AND NOT (
			role = ?
			AND ? != ?
			AND (SELECT COUNT(*) FROM users WHERE role = ?) <= 1
		  );
	`, user.Username, user.Password, user.Email, user.Role, user.ID, RoleAdmin, user.Role, RoleAdmin, RoleAdmin)
	if err != nil {
		return f.mapConstraintViolationToError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 && existingUser.Role == RoleAdmin && user.Role != RoleAdmin {
		return ErrLastAdminCannotBeDemoted
	}
	return nil
}

func (f *UserStore) DeleteUser(id string) error {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return err
	}

	// Check if a user with the given ID exists
	_, err = f.GetUserByID(id)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// Delete the user from the database
	_, err = f.db.Exec(`
		DELETE FROM users
		WHERE id = ?;
	`, id)
	if err != nil {
		return err
	}
	return nil
}

func (f *UserStore) GetAdminUser() (*User, error) {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return nil, err
	}
	// Query the admin user
	row := f.db.QueryRow(`SELECT ` + userColumns + `
		FROM users
		WHERE role = 'admin'
		LIMIT 1;
	`)

	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (f *UserStore) GetAllUsers() ([]*User, error) {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return nil, err
	}
	// Query all users
	rows, err := f.db.Query(`SELECT ` + userColumns + `
		FROM users;
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var users []*User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (f *UserStore) CountAdminUsers() (int, error) {
	err := f.Connect()
	if err != nil {
		return 0, err
	}
	row := f.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin';`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// CountEditorUsers counts users with role admin or editor — the same
// "editor" definition used across the plan-tier pricing (viewers are always
// unlimited, admin+editor together count against the plan's editor limit).
func (f *UserStore) CountEditorUsers() (int, error) {
	err := f.Connect()
	if err != nil {
		return 0, err
	}
	row := f.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role IN ('admin', 'editor');`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (f *UserStore) GetUserCount() (int, error) {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return 0, err
	}
	// Query the user count
	row := f.db.QueryRow(`
		SELECT COUNT(*)
		FROM users;
	`)
	var count int
	err = row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (f *UserStore) mapConstraintViolationToError(err error) error {
	// Check if the error is a constraint violation

	if err, ok := err.(interface{ Error() string }); ok {
		msg := err.Error()
		// Check for unique constraint violation
		if strings.Contains(msg, "UNIQUE constraint failed: users.username") {
			return ErrUserAlreadyExists
		}

		if strings.Contains(msg, "UNIQUE constraint failed: users.email") {
			return ErrUserAlreadyExists
		}
	}
	return err
}

func (f *UserStore) UpdatePassword(userID string, newPassword string) error {
	// Ensure the database is connected
	err := f.Connect()
	if err != nil {
		return err
	}

	// Check if a user with the given ID exists
	_, err = f.GetUserByID(userID)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// Update the user's password in the database
	_, err = f.db.Exec(`
		UPDATE users
		SET password = ?, must_change_password = 0
		WHERE id = ?;
	`, newPassword, userID)
	if err != nil {
		return err
	}
	return nil
}
func (f *UserStore) SetMustSetPassword(userID string, value bool) error {
	if err := f.Connect(); err != nil {
		return err
	}
	if _, err := f.GetUserByID(userID); err != nil {
		return err
	}

	v := 0
	if value {
		v = 1
	}
	_, err := f.db.Exec(`
		UPDATE users
		SET must_set_password = ?
		WHERE id = ?;
	`, v, userID)
	return err
}

// ensureCredentialSchema removes obsolete credential columns and keeps account rows intact.
func (f *UserStore) ensureCredentialSchema() error {
	columns, err := f.existingColumns()
	if err != nil {
		return err
	}
	if !columns["must_change_password"] {
		if _, err := f.db.Exec("ALTER TABLE users ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
	}
	for column := range columns {
		if strings.HasPrefix(column, "totp_") {
			if _, err := f.db.Exec(`ALTER TABLE users DROP COLUMN "` + column + `"`); err != nil {
				return err
			}
		}
	}
	return nil
}
