package auth

import (
	"database/sql"
	"log/slog"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

const logCloseRowsFailed = "could not close rows"

type UserStore struct {
	mu         sync.Mutex
	storageDir string
	db         *sql.DB
	// suspended is set by suspend() and makes Connect() refuse to lazily
	// reopen db — see suspend's doc comment.
	suspended bool
}

// NewUserStore opens the configured MySQL account store.
func NewUserStore(storageDir string) (*UserStore, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &UserStore{storageDir: storageDir, db: db}, nil
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
	db, err := database.Open(f.storageDir)
	if err != nil {
		return err
	}
	f.db = db
	return nil
}

// suspend blocks account access while a restore replaces business records.
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
			AND (SELECT COUNT(*) FROM (SELECT id FROM users WHERE role = ?) AS administrators) <= 1
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

// mapConstraintViolationToError preserves the public duplicate-account error.
func (f *UserStore) mapConstraintViolationToError(err error) error {
	if database.IsDuplicate(err) {
		return ErrUserAlreadyExists
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
