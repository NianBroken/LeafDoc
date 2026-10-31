package auth

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// SessionStore 保存令牌会话记录，通过互斥锁串行访问当前数据库句柄。
// 构造时启动每小时清理过期记录的协程，Close 取消并等待其退出后关闭连接。
type SessionStore struct {
	mu         sync.Mutex
	storageDir string
	db         *sql.DB
	cancel     context.CancelFunc
	done       chan struct{}
	log        *slog.Logger
}

// NewSessionStore 打开存储并执行连通检查，成功后才启动定时清理任务。
// 初始化失败取消内部上下文并返回错误，不返回可用存储对象。
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

	// 计时器第一次触发在一小时之后，失败记录告警，后续周期继续执行。
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

// withDB 持有互斥锁直到数据库回调结束，句柄为空时按存储目录重新打开。
// 此处没有单独的已关闭标记，关闭后的再次调用仍会尝试建立连接。
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

// ensureSchema 复用数据库层初始化后执行 Ping，此方法本身不包含建表语句。
func (s *SessionStore) ensureSchema() error {
	return s.withDB(func(db *sql.DB) error { return db.Ping() })
}

// Close 先停止清理协程，再在数据库互斥锁内关闭句柄。
// 已关闭的 done 通道可重复接收，数据库句柄只有关闭成功后才设为空。
func (s *SessionStore) Close() error {
	// 取消上下文使周期循环从 select 退出。
	s.cancel()
	// 等待后台清理完成，避免关闭连接时后台任务仍在执行数据库操作。
	<-s.done
	// 与其他数据库访问串行关闭，错误交给调用方。
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

// CreateSession 插入会话标识、用户、类型及 Unix 秒时间，初始撤销时间为空。
// 此入口不验证类型或到期时间，也不将重复标识转换为更新。
func (s *SessionStore) CreateSession(id, userID, tokenType string, expiresAt time.Time) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO sessions (id, user_id, token_type, created_at, expires_at, revoked_at)
			VALUES (?, ?, ?, ?, ?, NULL);
		`, id, userID, tokenType, time.Now().Unix(), expiresAt.Unix())
		return err
	})
}

// IsActive 要求标识、用户与类型同时匹配，记录不存在或已撤销返回 false。
// 时间按秒比较，仅 now 严格晚于到期秒才视为过期，相等时仍返回有效。
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
		// 不存在记录是正常的无效会话结果，不作为数据库故障向上返回。
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

// RevokeSession 只给尚未撤销的标识写入撤销时间，不要求该记录尚未到期。
// 没有匹配记录时 SQL 仍可成功，不检查受影响行数。
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

// RevokeAllSessionsForUser 标记该用户所有尚未撤销的记录，不删除行或修改令牌签名。
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

// RevokeAllSessionsForUserExcept 撤销指定用户中标识不等于 exceptID 的未撤销记录。
// 过滤条件不检查有效期，空排除标识不会匹配正常生成的非空会话标识。
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

// DeleteAllSessions 直接清空会话表，后续刷新校验不能再找到原持久化记录。
func (s *SessionStore) DeleteAllSessions() error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`DELETE FROM sessions;`)
		return err
	})
}

// CleanupExpiredSessions 删除到期秒小于或等于当前秒的记录，不按撤销时间清理未到期行。
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

// Generation 读取固定主键的认证代际，记录缺失时用数据库 UUID 函数创建后重新读取。
// 插入竞争时保留已有代际，不因重新打开存储而轮换该值。
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
