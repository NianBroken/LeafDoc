package auth

import (
	"errors"
	"log/slog"
	"sync"

	"golang.org/x/crypto/bcrypt"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

type AuthService struct {
	// mu guards the account service handle during restoration.
	mu          sync.RWMutex
	userService *UserService
	sessions    *SessionManager
	attempts    *loginAttemptTracker
	dummyHash   []byte
	log         *slog.Logger
}

// users returns the current *UserService under a read lock. Callers use the
// returned pointer directly for the rest of their operation — the lock is
// only held long enough to copy the pointer, never across a query or a
// bcrypt hash, so ReplaceUserStore swapping it mid-request never serializes
// unrelated logins/refreshes on this mutex.
func (a *AuthService) users() *UserService {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.userService
}

// UserService returns the active account service after any restoration.
func (a *AuthService) UserService() *UserService {
	return a.users()
}

// ReplaceUserStore replaces the account connection after MySQL restoration.
func (a *AuthService) ReplaceUserStore(storageDir string) error {
	newStore, err := NewUserStore(storageDir)
	if err != nil {
		return err
	}
	newUserService := NewUserService(newStore)

	a.mu.Lock()
	old := a.userService
	// Preserve the plan's editor limit across the swap — a restore changes
	// user data, not which plan tier this instance is running under.
	newUserService.SetEditorLimit(old.editorLimit)
	a.userService = newUserService
	a.mu.Unlock()

	if old != nil {
		if err := old.Close(); err != nil {
			a.log.Warn("failed to close previous user store after restore", "error", err)
		}
	}
	return nil
}

// PauseUserStoreForSwap blocks account access during restoration.
func (a *AuthService) PauseUserStoreForSwap() error {
	return a.users().suspendStore()
}

// InvalidateAllSessions requires fresh authentication after restored accounts are committed.
func (a *AuthService) InvalidateAllSessions() error {
	return a.sessions.sessionStore.DeleteAllSessions()
}

func NewAuthService(userService *UserService, sessions *SessionManager) *AuthService {
	// Pre-compute a dummy hash to equalize Login() timing for non-existent users,
	// preventing username enumeration via response-time differences.
	dummyHash, _ := bcrypt.GenerateFromPassword([]byte("leafdoc-dummy-password"), bcrypt.DefaultCost)
	a := &AuthService{
		userService: userService,
		sessions:    sessions,
		attempts:    newLoginAttemptTracker(),
		dummyHash:   dummyHash,
		log:         slog.Default().With("component", "AuthService"),
	}
	// Wired here rather than passed into NewSessionManager: a.users() reads
	// through the mutex ReplaceUserStore swaps, so RefreshToken/ValidateToken
	// stay correct across a live-restore hot-swap — this closure calls
	// a.users() fresh on every invocation rather than capturing one
	// *UserService at construction time.
	sessions.resolveUser = func(id string) (*User, error) {
		return a.users().GetUserByID(id)
	}
	return a
}

func (a *AuthService) Close() error {
	var errs []error

	if a.sessions != nil {
		if err := a.sessions.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if users := a.users(); users != nil {
		if err := users.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

type AuthToken struct {
	Token                string      `json:"token"`
	RefreshToken         string      `json:"refresh_token"`
	AccessTokenExpiresAt int64       `json:"accessTokenExpiresAt"`
	User                 *PublicUser `json:"user"`
}

func (a *AuthService) Login(identifier, password string) (*AuthToken, error) {
	user, err := a.users().GetUserByIdentifier(identifier)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(a.dummyHash, []byte(password))
		a.log.Warn("login failed: invalid credentials")
		return nil, ErrUserInvalidCredentials
	}

	if !a.attempts.recordAttempt(user.ID) {
		a.log.Warn("login rejected: account locked", "userID", user.ID)
		return nil, ErrUserAccountLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		a.log.Warn("login failed: invalid credentials")
		return nil, ErrUserInvalidCredentials
	}

	user.Password = ""

	a.attempts.reset(user.ID)
	a.log.Info("login succeeded", "userID", user.ID)
	return a.sessions.IssueSession(user)
}
func (a *AuthService) IssueSessionForUser(userID string) (*AuthToken, error) {
	user, err := a.users().GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return a.sessions.IssueSession(user)
}

// errUserStoreUnavailable is returned while the user store is suspended for
// an in-progress live restore (see UserStore.suspend / PauseUserStoreForSwap).
// A GET request landing in that window gets this immediately instead of
// racing a reconnect against the file swap.
func errUserStoreUnavailable() error {
	return sharederrors.NewLocalizedError(
		ErrCodeAuthUserStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"user store is suspended for an in-progress restore",
		nil,
	)
}

func (a *AuthService) RefreshToken(refreshToken string) (*AuthToken, error) {
	return a.sessions.RefreshToken(refreshToken)
}

func (a *AuthService) RevokeRefreshToken(tokenString string) error {
	return a.sessions.RevokeRefreshToken(tokenString)
}

func (a *AuthService) RevokeAllUserSessions(userID string) error {
	return a.sessions.RevokeAllUserSessions(userID)
}

func (a *AuthService) RevokeAllUserSessionsExceptCurrent(userID, currentRefreshToken string) error {
	return a.sessions.RevokeAllUserSessionsExceptCurrent(userID, currentRefreshToken)
}

func (a *AuthService) ValidateToken(tokenString string) (*User, error) {
	return a.sessions.ValidateToken(tokenString)
}
