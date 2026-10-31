// 本文件保存邀请和重置密码邮件使用的一次性令牌。数据库保存密钥摘要、用途、
// 有效期和使用时间；链接中的完整令牌只由签发入口返回，不写入此表。
package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// 两个用途分别用于密码重置和邀请。Resolve 比较用途原值，阻止跨用途使用令牌。
const (
	PurposePasswordReset = "password_reset"
	PurposeInvite        = "invite"
)

// 邮件业务采用的有效期分别为一小时和七天。Issue 接收调用方传入的时长，
// 不在存储层强制限定为这两个常量，也不读取运维配置。
const (
	PasswordResetTokenTTL = time.Hour
	InviteTokenTTL        = 7 * 24 * time.Hour
)

// 查询不到标识时仍执行密钥摘要和常量时间比较，避免省略这段计算。
// 该处理不保证数据库查询及整个请求的耗时相同。
var dummyEmailTokenHash = hashSecret("leafdoc-dummy-email-token-secret-for-timing-equalization")

// EmailToken 对应数据库中的邮件令牌记录。ConsumedAt 为空表示尚未使用，
// tokenHash 保存密钥文本的 SHA-256 十六进制摘要，不保存链接中的明文密钥。
type EmailToken struct {
	ID         string
	UserID     string
	Purpose    string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ConsumedAt *time.Time

	tokenHash string
}

// EmailTokenStore 串行执行本对象上的数据库操作，并持有每小时清理过期记录的协程。
// 互斥锁覆盖整个数据库回调，关闭连接也使用同一把锁。
type EmailTokenStore struct {
	mu         sync.Mutex
	storageDir string
	db         *sql.DB
	cancel     context.CancelFunc
	done       chan struct{}
	log        *slog.Logger
}

// NewEmailTokenStore 先打开并探测数据库，再启动定时清理。初始化失败时取消上下文，
// 不启动后台协程；成功后调用方需要通过 Close 停止协程并关闭连接。
func NewEmailTokenStore(storageDir string) (*EmailTokenStore, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &EmailTokenStore{
		storageDir: storageDir,
		cancel:     cancel,
		done:       make(chan struct{}),
		log:        slog.Default().With("component", "EmailTokenStore"),
	}

	err := s.ensureSchema()
	if err != nil {
		cancel()
		return nil, err
	}

	// 清理失败记录警告并继续等待后续时钟事件，取消时停止计时器并关闭完成通道。
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.CleanupExpired(); err != nil {
					s.log.Warn("failed to cleanup expired email tokens", "error", err)
				}
			}
		}
	}()

	return s, nil
}

// withDB 在锁内按需打开连接并执行回调，回调错误原样返回。
// Close 清空连接后仍可由此重新打开，存储没有永久关闭状态标记。
func (s *EmailTokenStore) withDB(fn func(db *sql.DB) error) error {
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

// ensureSchema 通过统一连接入口执行 Ping，本函数自身没有建表语句。
func (s *EmailTokenStore) ensureSchema() error {
	return s.withDB(func(db *sql.DB) error { return db.Ping() })
}

// Close 取消定时清理并等到协程退出，再关闭数据库连接。
// 关闭连接失败时保留连接指针并返回错误；成功时清空指针。
func (s *EmailTokenStore) Close() error {
	s.cancel()
	<-s.done

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

// Issue 为账号生成标识和随机密钥，将密钥摘要与用途、有效期写入数据库。
// 写入成功后返回 id.secret，随机生成或写入失败则返回空文本和错误。
// 本入口不检查账号是否存在，不验证用途名称或有效期正负值。
func (s *EmailTokenStore) Issue(userID, purpose string, ttl time.Duration) (rawToken string, err error) {
	id, err := shared.GenerateUniqueID()
	if err != nil {
		return "", err
	}
	secret, err := generateSecret()
	if err != nil {
		return "", err
	}
	hash := hashSecret(secret)

	// 时间按 Unix 秒保存，读取时不会恢复原时间的亚秒部分。
	now := time.Now()
	err = s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO email_tokens (id, token_hash, user_id, purpose, created_at, expires_at, consumed_at)
			VALUES (?, ?, ?, ?, ?, ?, NULL);
		`, id, hash, userID, purpose, now.Unix(), now.Add(ttl).Unix())
		return err
	})
	if err != nil {
		return "", err
	}

	return id + "." + secret, nil
}

// Resolve 按令牌标识查询，核对密钥摘要、用途、使用状态和有效期，然后返回记录。
// 格式错误、未知标识、密钥错误及业务条件不符均返回 ErrEmailTokenInvalid，
// 其他数据库错误原样返回。成功解析不会消耗令牌，使用方另行调用 Consume。
func (s *EmailTokenStore) Resolve(rawToken, purpose string) (*EmailToken, error) {
	id, secret, ok := parseEmailToken(rawToken)
	if !ok {
		return nil, ErrEmailTokenInvalid
	}

	tok, err := s.getByID(id)
	found := err == nil
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// 未命中记录仍比较占位摘要；只有命中记录且摘要相等才继续检查业务条件。
	storedHash := dummyEmailTokenHash
	if found {
		storedHash = tok.tokenHash
	}
	match := subtle.ConstantTimeCompare([]byte(hashSecret(secret)), []byte(storedHash)) == 1
	if !found || !match {
		return nil, ErrEmailTokenInvalid
	}

	// 用途先于使用状态和有效期检查，任意一个条件不符都拒绝该令牌。
	if tok.Purpose != purpose {
		return nil, ErrEmailTokenInvalid
	}
	if tok.ConsumedAt != nil {
		return nil, ErrEmailTokenInvalid
	}
	// 严格晚于到期时间才判过期，此判断与 CleanupExpired 的秒级删除条件不同。
	if time.Now().After(tok.ExpiresAt) {
		return nil, ErrEmailTokenInvalid
	}

	return tok, nil
}

// Consume 只把指定标识的未使用记录标记为已使用，通过 SQL 条件保证并发请求中
// 只有一个请求能更新该行。找不到记录或已经使用时返回 ErrEmailTokenInvalid。
// 本入口不核对密钥、用途或到期时间，这些检查由 Resolve 完成。
func (s *EmailTokenStore) Consume(id string) error {
	var rowsAffected int64
	err := s.withDB(func(db *sql.DB) error {
		result, err := db.Exec(`
			UPDATE email_tokens SET consumed_at = ? WHERE id = ? AND consumed_at IS NULL;
		`, time.Now().Unix(), id)
		if err != nil {
			return err
		}
		rowsAffected, err = result.RowsAffected()
		return err
	})
	if err != nil {
		return err
	}
	// 只有恰好更新一行才确认消费成功，数据库执行和行数读取错误分别向上传递。
	if rowsAffected != 1 {
		return ErrEmailTokenInvalid
	}
	return nil
}

// CleanupExpired 删除到期秒数小于或等于当前秒的所有记录，不区分是否已经使用。
func (s *EmailTokenStore) CleanupExpired() error {
	now := time.Now()
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`DELETE FROM email_tokens WHERE expires_at <= ?;`, now.Unix())
		return err
	})
}

// getByID 读取原始记录并把秒级时间转成 time.Time；未消费字段保留空指针。
// 没有记录时保留 sql.ErrNoRows，交给 Resolve 区分未知标识和其他数据库失败。
func (s *EmailTokenStore) getByID(id string) (*EmailToken, error) {
	tok := &EmailToken{}
	err := s.withDB(func(db *sql.DB) error {
		var createdAt, expiresAt int64
		var consumedAt sql.NullInt64
		err := db.QueryRow(`
			SELECT id, token_hash, user_id, purpose, created_at, expires_at, consumed_at
			FROM email_tokens WHERE id = ?;
		`, id).Scan(&tok.ID, &tok.tokenHash, &tok.UserID, &tok.Purpose, &createdAt, &expiresAt, &consumedAt)
		if err != nil {
			return err
		}
		tok.CreatedAt = time.Unix(createdAt, 0)
		tok.ExpiresAt = time.Unix(expiresAt, 0)
		if consumedAt.Valid {
			t := time.Unix(consumedAt.Int64, 0)
			tok.ConsumedAt = &t
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tok, nil
}

// parseEmailToken 只按第一个点分隔标识和密钥，并要求两部分非空。
// 它不裁剪空白，也不检查标识格式、密钥长度或密钥中是否还有点。
func parseEmailToken(raw string) (id, secret string, ok bool) {
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
