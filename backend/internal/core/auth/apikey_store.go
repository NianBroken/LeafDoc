// 本文件负责 API 密钥记录的数据库映射、查询、撤销和使用时间写入。
// 令牌解析、摘要核对和账号权限判断位于 APIKeyService，不由存储层执行。
package auth

import (
	"database/sql"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// APIKey 表示一条持久化密钥记录。KeyHash 保存密钥文本的 SHA-256 摘要，
// 公开前缀用于查询，记录中没有可直接拼回完整令牌的明文密钥。
type APIKey struct {
	ID         string
	Name       string
	UserID     string // 密钥所属账号，也是认证成功后的请求主体。
	Prefix     string // 公开的查询前缀，不承担密钥校验职责。
	KeyHash    string // 密钥文本的 SHA-256 十六进制摘要。
	Role       string // 由服务层与账号角色比较，限制有效权限。
	ExpiresAt  *time.Time
	CreatedBy  string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

// IsActive 只检查撤销与有效期，等于到期时刻即失效。
// 它不验证密钥内容、所属账号是否存在或账号权限。
func (k *APIKey) IsActive(now time.Time) bool {
	if k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && !now.Before(*k.ExpiresAt) {
		return false
	}
	return true
}

// APIKeyStore 串行执行本对象的数据库回调，锁同时保护连接和恢复期间的停用状态。
type APIKeyStore struct {
	mu         sync.Mutex
	storageDir string
	db         *sql.DB
	// 停用后 withDB 直接返回不可用错误，不再按需打开连接。
	suspended bool
}

// NewAPIKeyStore 通过统一连接入口打开数据库并执行连通性检查。
func NewAPIKeyStore(storageDir string) (*APIKeyStore, error) {
	s := &APIKeyStore{
		storageDir: storageDir,
	}

	if err := s.ensureSchema(); err != nil {
		return nil, err
	}

	return s, nil
}

// withDB 在持锁期间检查停用标记、按需打开连接并执行完整回调。
// 回调错误原样返回，未停用对象在 Close 后仍可重新建立连接。
func (s *APIKeyStore) withDB(fn func(db *sql.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.suspended {
		return errAPIKeyStoreUnavailable()
	}

	if s.db == nil {
		db, err := database.Open(s.storageDir)
		if err != nil {
			return err
		}
		s.db = db
	}

	return fn(s.db)
}

// suspend 停用当前存储对象并关闭连接，恢复流程通过服务层替换存储对象继续访问。
// 即使关闭连接返回错误，也清空指针并保留停用状态。
func (s *APIKeyStore) suspend() error {
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

// ErrCodeAPIKeyStoreUnavailable 统一标识恢复期间的密钥存储不可用错误。
// 服务层和 HTTP 层使用同一代码识别此状态，避免把它当作密钥无效。
const ErrCodeAPIKeyStoreUnavailable = "apikey_store_unavailable"

// errAPIKeyStoreUnavailable 构造停用状态对应的本地化错误，供 withDB 拒绝读写。
func errAPIKeyStoreUnavailable() error {
	return sharederrors.NewLocalizedError(
		ErrCodeAPIKeyStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"api key store is suspended for an in-progress restore",
		nil,
	)
}

// ensureSchema 仅经统一连接入口执行 Ping，本函数自身不包含建表语句。
func (s *APIKeyStore) ensureSchema() error {
	return s.withDB(func(db *sql.DB) error { return db.Ping() })
}

// Close 在锁内关闭连接，成功后清空指针，不改变停用标记。
// 关闭失败时保留连接指针并返回错误。
func (s *APIKeyStore) Close() error {
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

// CreateAPIKey 按传入记录写入所有字段，可空时间映射为 SQL NULL。
// 此入口不生成随机值或检查角色，重复键错误统一映射为前缀冲突。
func (s *APIKeyStore) CreateAPIKey(key *APIKey) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO api_keys (id, name, user_id, prefix, key_hash, role, expires_at, created_by, created_at, last_used_at, revoked_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`, key.ID, key.Name, key.UserID, key.Prefix, key.KeyHash, key.Role,
			timeToNullInt64(key.ExpiresAt), key.CreatedBy, key.CreatedAt.Unix(),
			timeToNullInt64(key.LastUsedAt), timeToNullInt64(key.RevokedAt))
		if err != nil {
			return s.mapConstraintViolationToError(err)
		}
		return nil
	})
}

// GetByPrefix 按公开前缀读取记录，不排除过期或撤销的密钥。
// 没有匹配行时返回 ErrAPIKeyNotFound，其他读取错误原样返回。
func (s *APIKeyStore) GetByPrefix(prefix string) (*APIKey, error) {
	var key *APIKey
	err := s.withDB(func(db *sql.DB) error {
		row := db.QueryRow(`
			SELECT id, name, user_id, prefix, key_hash, role, expires_at, created_by, created_at, last_used_at, revoked_at
			FROM api_keys
			WHERE prefix = ?;
		`, prefix)
		var scanErr error
		key, scanErr = scanAPIKey(row)
		return scanErr
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return key, nil
}

// GetByID 按记录标识读取与前缀查询相同的完整字段，供撤销前确认记录存在。
func (s *APIKeyStore) GetByID(id string) (*APIKey, error) {
	var key *APIKey
	err := s.withDB(func(db *sql.DB) error {
		row := db.QueryRow(`
			SELECT id, name, user_id, prefix, key_hash, role, expires_at, created_by, created_at, last_used_at, revoked_at
			FROM api_keys
			WHERE id = ?;
		`, id)
		var scanErr error
		key, scanErr = scanAPIKey(row)
		return scanErr
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return key, nil
}

// ListAll 按创建时间降序返回全部密钥，包括已经过期或撤销的记录。
// 时间相同的行没有额外排序条件，查询或遍历失败时不返回部分结果。
func (s *APIKeyStore) ListAll() ([]*APIKey, error) {
	var keys []*APIKey
	err := s.withDB(func(db *sql.DB) error {
		rows, err := db.Query(`
			SELECT id, name, user_id, prefix, key_hash, role, expires_at, created_by, created_at, last_used_at, revoked_at
			FROM api_keys
			ORDER BY created_at DESC;
		`)
		if err != nil {
			return err
		}
		// 无论扫描成功或失败都关闭游标，此处没有把关闭错误纳入返回值。
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			key, err := scanAPIKey(rows)
			if err != nil {
				return err
			}
			keys = append(keys, key)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Revoke 先确认记录存在，再只对尚未撤销的行写入当前秒数，保留已有撤销时间。
// 查询和写入是两次独立操作，更新不检查受影响行数；这里不删除记录或账号会话。
func (s *APIKeyStore) Revoke(id string) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			UPDATE api_keys
			SET revoked_at = ?
			WHERE id = ? AND revoked_at IS NULL;
		`, time.Now().Unix(), id)
		return err
	})
}

// DeleteAllForUser 删除指定账号的全部密钥记录，没有匹配行也不返回专用错误。
// APIKeyService 认证时另行读取所属账号，账号不存在会使认证失败。
func (s *APIKeyStore) DeleteAllForUser(userID string) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`DELETE FROM api_keys WHERE user_id = ?;`, userID)
		return err
	})
}

// TouchLastUsed 按秒写入最后使用时间，不检查撤销状态，也不限制写入频率。
// 服务层决定是否节流，找不到记录时不会根据受影响行数生成错误。
func (s *APIKeyStore) TouchLastUsed(id string, at time.Time) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			UPDATE api_keys
			SET last_used_at = ?
			WHERE id = ?;
		`, at.Unix(), id)
		return err
	})
}

// mapConstraintViolationToError 把所有数据库重复键错误归为前缀冲突，
// 不在此区分具体约束名称；其他错误保留原值。
func (s *APIKeyStore) mapConstraintViolationToError(err error) error {
	if database.IsDuplicate(err) {
		return ErrAPIKeyPrefixCollision
	}
	return err
}

// rowScanner 让单行查询与多行遍历共用字段读取方法。
type rowScanner interface {
	Scan(dest ...any) error
}

// scanAPIKey 按各查询共有的列顺序填充记录，再还原秒级时间。
// 任意字段读取失败时返回空对象指针，防止调用方使用部分记录。
func scanAPIKey(row rowScanner) (*APIKey, error) {
	key := &APIKey{}
	var expiresAt, lastUsedAt, revokedAt sql.NullInt64
	var createdAt int64

	if err := row.Scan(&key.ID, &key.Name, &key.UserID, &key.Prefix, &key.KeyHash, &key.Role,
		&expiresAt, &key.CreatedBy, &createdAt, &lastUsedAt, &revokedAt); err != nil {
		return nil, err
	}

	key.CreatedAt = time.Unix(createdAt, 0)
	key.ExpiresAt = nullInt64ToTime(expiresAt)
	key.LastUsedAt = nullInt64ToTime(lastUsedAt)
	key.RevokedAt = nullInt64ToTime(revokedAt)
	return key, nil
}

// timeToNullInt64 将空指针编码为 SQL NULL，其他时间只保存 Unix 秒。
func timeToNullInt64(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}

// nullInt64ToTime 将 SQL NULL 还原为空指针，零秒和其他有效数值均保留为时间。
func nullInt64ToTime(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.Unix(v.Int64, 0)
	return &t
}
