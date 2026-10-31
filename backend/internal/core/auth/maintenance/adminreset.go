// Package maintenance 提供管理员恢复操作，直接使用现有用户和会话存储。
package maintenance

import (
	"fmt"
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
)

// ResetAdminPassword 先通过用户服务设置管理员信息和密码，再撤销该用户的全部会话。
// 两阶段不在同一事务中，会话存储打开或撤销失败时密码操作已经完成。
// 用户与会话连接均延迟关闭，关闭失败记录到默认日志器，不覆盖主要返回结果。
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
