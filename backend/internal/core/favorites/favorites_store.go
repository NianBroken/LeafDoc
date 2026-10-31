// Package favorites 保存按用户隔离的页面收藏关系。
// 收藏由用户操作写入 MySQL，不从页面正文或页面树推导。查询以用户 ID 为条件，
// 页面删除和用户删除分别通过专用方法清理相关关系。
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

// ErrCodeFavoritesStoreUnavailable 标识收藏连接暂停或未打开的错误，供上层选择本地化提示。
const ErrCodeFavoritesStoreUnavailable = "favorites_store_unavailable"

// errFavoritesStoreUnavailable 构造带稳定错误码的恢复期间不可用错误。
// 对外消息与内部诊断分别保存在本地化错误对象中，此函数不执行重试。
func errFavoritesStoreUnavailable() error {
	return sharederrors.NewLocalizedError(
		ErrCodeFavoritesStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"favorites store is suspended for an in-progress restore",
		nil,
	)
}

// FavoritesStore 持有收藏表访问所用连接或外部执行器。
// 普通连接上的操作由互斥锁串行化，外部执行器由其调用方管理生命周期。
type FavoritesStore struct {
	executor database.Executor
	mu       sync.Mutex
	db       *sql.DB
	// suspended 阻止普通连接继续服务，Replace 成功接入新连接后解除暂停。
	// WithExecutor 创建的视图直接使用外部执行器，不经过该状态检查。
	suspended bool
	log       *slog.Logger
}

// NewFavoritesStore 根据存储目录对应的数据库配置打开连接。
// 打开失败直接返回错误，成功时由返回的存储对象承担连接关闭职责。
func NewFavoritesStore(storageDir string, log *slog.Logger) (*FavoritesStore, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &FavoritesStore{db: db, log: log}, nil
}

// withDB 在持锁期间检查连接并运行回调，使查询与连接替换互斥。
// 暂停或空连接直接返回不可用错误，回调错误原样向调用方传递。
func (s *FavoritesStore) withDB(fn func(db *sql.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.suspended || s.db == nil {
		return errFavoritesStoreUnavailable()
	}
	return fn(s.db)
}

// Add 写入用户与页面的收藏关系，并以当前 UTC 时间记录收藏时间。
// 唯一键已存在时保留原时间，因此重复调用不会把旧收藏移动到列表前部。
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

// Remove 只删除指定用户对指定页面的收藏，不影响其他用户。
// SQL 不要求存在匹配行，没有该收藏时也返回成功。
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

// ListPageIDsForUser 按收藏时间降序返回指定用户的页面 ID。
// 时间相同的记录没有额外排序条件，无匹配记录返回空切片值 nil。
// 查询、扫描和遍历错误向上传递，结果集关闭失败交由共享日志函数记录。
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

// DeleteAllForPage 按页面 ID 清理全部用户的对应收藏，供页面删除流程调用。
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

// DeleteAllForUser 清理指定用户的全部收藏，供用户删除流程调用。
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

// PauseForSwap 在锁内设置暂停状态并关闭原连接。
// 即使关闭返回错误，连接字段仍置空，后续普通查询不会继续使用原连接。
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

// Replace 先打开新连接，成功后在锁内替换连接并恢复访问。
// 打开失败保持原状态；原连接在锁外关闭，关闭失败仅在已提供日志器时记录。
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

// Close 关闭本对象持有的连接，外部执行器不在此处关闭。
// 关闭成功后清空连接字段，关闭失败保留字段并返回错误。
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

// WithExecutor 创建使用外部执行器的独立视图，常用于把收藏清理纳入页面事务。
// 视图沿用日志器，不复制连接及暂停状态，也不负责提交或关闭外部执行器。
func (s *FavoritesStore) WithExecutor(executor database.Executor) *FavoritesStore {
	return &FavoritesStore{executor: executor, log: s.log}
}

// withExecutor 优先使用外部执行器；未绑定时才走普通连接的锁与暂停检查。
func (s *FavoritesStore) withExecutor(fn func(database.Executor) error) error {
	if s.executor != nil {
		return fn(s.executor)
	}
	return s.withDB(func(db *sql.DB) error { return fn(db) })
}
