package maintenance

import (
	"fmt"
	"log/slog"

	"github.com/perber/wiki/internal/core/auth"
)

// ResetAdminPassword updates an administrator and revokes that account's sessions.
func ResetAdminPassword(storageDir, username, email string, password ...string) (*auth.User, error) {
	store, err := auth.NewUserStore(storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open user store: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Default().Error("could not close store", "error", err)
		}
	}()

	userService := auth.NewUserService(store)
	user, err := userService.ResetAdminUserPassword(username, email, password...)
	if err != nil {
		return nil, err
	}
	sessions, err := auth.NewSessionStore(storageDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := sessions.Close(); err != nil {
			slog.Default().Error("could not close sessions", "error", err)
		}
	}()
	if err := sessions.RevokeAllSessionsForUser(user.ID); err != nil {
		return nil, err
	}
	return user, nil
}
