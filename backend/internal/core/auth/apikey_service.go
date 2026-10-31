// 本文件编排 API 密钥的生成、存储访问和请求认证。完整密钥由公开前缀与随机密钥
// 组成，认证时读取账号的当前角色，并将有效角色限制在账号与密钥两者的较低权限。
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// 未查到前缀时仍执行密钥摘要和常量时间比较，避免直接省略这一段计算。
// 该处理不保证数据库查询、日志写入或整个认证过程的执行时间相同。
var dummySecretHash = hashSecret("leafdoc-dummy-api-key-secret-for-timing-equalization")

// hashSecret 对密钥文本计算 SHA-256，并输出固定长度的十六进制摘要。
// 生成器返回随机字节的十六进制文本，这里散列的是该文本的字节，不是解码后的随机值。
// 邮件令牌与 API 密钥共用此转换，认证时比较摘要而不存储明文密钥。
func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// API 密钥使用固定开头，供 Bearer 中间件识别凭据种类并选择认证入口。
const apiKeyTokenPrefix = "lw_"

// 最后使用时间超过五分钟才再次写入。该判断使用读取到的记录，
// 不在服务层为并发请求建立全局节流锁。
const apiKeyLastUsedThrottle = 5 * time.Minute

// APIKeyService 持有可替换的密钥存储，账号信息始终通过认证服务的当前账号入口读取。
type APIKeyService struct {
	// 读写锁只保护存储指针，查询和存储关闭不在这把锁内执行。
	mu    sync.RWMutex
	store *APIKeyStore
	// 通过认证服务解析密钥所属账号及其当前角色。
	authService *AuthService
	log         *slog.Logger
}

// NewAPIKeyService 组合存储与认证服务，并为操作失败日志设置组件名称。
func NewAPIKeyService(store *APIKeyStore, authService *AuthService) *APIKeyService {
	return &APIKeyService{store: store, authService: authService, log: slog.Default().With("component", "APIKeyService")}
}

// currentStore 在读锁内复制存储指针并立即释放锁。后续替换不会把已经取得的指针
// 自动指向另一个存储，当前操作继续使用自己取得的对象。
func (s *APIKeyService) currentStore() *APIKeyStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store
}

// Close 关闭调用时取得的密钥存储，并返回它的关闭结果。
func (s *APIKeyService) Close() error {
	return s.currentStore().Close()
}

// PauseForSwap 停用当前存储，供恢复流程阻止该对象继续接受密钥读写。
func (s *APIKeyService) PauseForSwap() error {
	return s.currentStore().suspend()
}

// Replace 先成功打开目标存储，再交换指针并关闭旧存储。打开失败时保留当前指针，
// 旧存储关闭失败只记警告，不撤销已完成的指针交换。
func (s *APIKeyService) Replace(storageDir string) error {
	newStore, err := NewAPIKeyStore(storageDir)
	if err != nil {
		return err
	}

	s.mu.Lock()
	old := s.store
	s.store = newStore
	s.mu.Unlock()

	if old != nil {
		if err := old.Close(); err != nil {
			s.log.Warn("failed to close previous api key store after restore", "error", err)
		}
	}
	return nil
}

// CreateAPIKeyParams 提供密钥名称、所属账号、角色、可选到期时间及创建者标识。
type CreateAPIKeyParams struct {
	Name      string
	UserID    string // 认证成功后以此账号为主体，并读取该账号的当前权限。
	Role      string // 限制密钥权限；空值在创建入口中使用只读角色。
	ExpiresAt *time.Time
	CreatedBy string // 随记录保存的创建者标识，此结构自身不验证创建者角色。
}

// CreateAPIKey 检查角色和所属账号后生成密钥记录，成功时返回记录及完整令牌。
// 数据库只保存密钥摘要，后续列表无法恢复明文。此处不验证名称、创建者或到期时间。
func (s *APIKeyService) CreateAPIKey(p CreateAPIKeyParams) (*APIKey, string, error) {
	// 空角色采用只读权限；其他角色必须属于认证模块接受的角色集合。
	role := p.Role
	if role == "" {
		role = RoleViewer
	}
	if !IsValidRole(role) {
		return nil, "", ErrUserInvalidRole
	}
	if _, err := s.authService.UserService().GetUserByID(p.UserID); err != nil {
		return nil, "", err
	}

	id, err := shared.GenerateUniqueID()
	if err != nil {
		return nil, "", err
	}

	secret, err := generateSecret()
	if err != nil {
		return nil, "", err
	}
	hash := hashSecret(secret)
	store := s.currentStore()

	// 前缀冲突时最多尝试两次，只重新生成前缀，记录标识和密钥摘要继续复用。
	// 其他写入错误立即返回，第二次仍冲突也直接返回错误。
	const maxAttempts = 2
	for attempt := 1; ; attempt++ {
		prefix, err := generatePrefix()
		if err != nil {
			return nil, "", err
		}

		key := &APIKey{
			ID:        id,
			Name:      p.Name,
			UserID:    p.UserID,
			Prefix:    prefix,
			KeyHash:   hash,
			Role:      role,
			ExpiresAt: p.ExpiresAt,
			CreatedBy: p.CreatedBy,
			CreatedAt: time.Now(),
		}

		if err := store.CreateAPIKey(key); err != nil {
			if err == ErrAPIKeyPrefixCollision && attempt < maxAttempts {
				continue
			}
			return nil, "", err
		}

		return key, apiKeyTokenPrefix + prefix + "_" + secret, nil
	}
}

// ListAPIKeys 委托当前存储读取密钥记录，不生成或返回完整令牌。
func (s *APIKeyService) ListAPIKeys() ([]*APIKey, error) {
	return s.currentStore().ListAll()
}

// RevokeAPIKey 将指定记录标记为撤销，保留数据库记录供列表和查询使用。
func (s *APIKeyService) RevokeAPIKey(id string) error {
	return s.currentStore().Revoke(id)
}

// DeleteAllForUser 删除指定账号拥有的全部密钥，供账号删除流程清理关联记录。
func (s *APIKeyService) DeleteAllForUser(userID string) error {
	return s.currentStore().DeleteAllForUser(userID)
}

// AsStoreUnavailableErr 识别密钥存储或账号存储在恢复期间不可用的本地化错误。
// 其他本地化错误也返回 false；认证服务和中间件据此保留不可用错误的独立含义。
func AsStoreUnavailableErr(err error) (*sharederrors.LocalizedError, bool) {
	loc, ok := sharederrors.AsLocalizedError(err)
	if !ok {
		return nil, false
	}
	if loc.Code == ErrCodeAPIKeyStoreUnavailable || loc.Code == ErrCodeAuthUserStoreUnavailable {
		return loc, true
	}
	return nil, false
}

// Resolve 接收不含 Bearer 方案名的完整密钥令牌，核对前缀、密钥、撤销和到期状态，
// 再读取所属账号。成功时复制账号，清空密码字段，并将角色限制为两者的较低权限。
// 格式错误、未知前缀和摘要不符归为无效密钥；撤销、过期及存储停用分别返回专用错误。
// 普通查询错误记警告后归为无效密钥，最后使用时间写入失败不阻止已通过的认证。
func (s *APIKeyService) Resolve(token string) (*User, error) {
	prefix, secret, ok := parseKeyToken(token)
	if !ok {
		return nil, ErrAPIKeyInvalid
	}

	store := s.currentStore()
	key, err := store.GetByPrefix(prefix)
	if _, ok := AsStoreUnavailableErr(err); ok {
		return nil, err
	}
	found := err == nil
	if err != nil && err != ErrAPIKeyNotFound {
		s.log.Warn("api key resolve: prefix lookup failed", "error", err)
	}

	// 查询失败仍计算摘要，但停用错误在上方直接返回，保留恢复期间不可用的含义。
	storedHash := dummySecretHash
	if found {
		storedHash = key.KeyHash
	}
	match := subtle.ConstantTimeCompare([]byte(hashSecret(secret)), []byte(storedHash)) == 1
	if !found || !match {
		return nil, ErrAPIKeyInvalid
	}

	// 只有正确密钥才检查撤销与到期，等于到期时刻也拒绝访问。
	now := time.Now()
	if key.RevokedAt != nil {
		return nil, ErrAPIKeyRevoked
	}
	if key.ExpiresAt != nil && !now.Before(*key.ExpiresAt) {
		return nil, ErrAPIKeyExpired
	}

	// 每次认证重新读取账号，账号删除或角色变化会反映到后续认证结果中。
	owner, err := s.authService.UserService().GetUserByID(key.UserID)
	if _, ok := AsStoreUnavailableErr(err); ok {
		return nil, err
	}
	if err != nil {
		s.log.Warn("api key resolve: owner lookup failed", "error", err, "keyID", key.ID)
		return nil, ErrAPIKeyInvalid
	}

	// 未记录使用时间，或距离上次记录严格超过节流时长时，尝试写入当前时间。
	if key.LastUsedAt == nil || now.Sub(*key.LastUsedAt) > apiKeyLastUsedThrottle {
		if err := store.TouchLastUsed(key.ID, now); err != nil {
			s.log.Warn("failed to update api key last_used_at", "error", err, "keyID", key.ID)
		}
	}

	effective := *owner
	effective.Role = intersectRole(owner.Role, key.Role)
	effective.Password = ""
	return &effective, nil
}

// 以下辅助函数负责令牌文本结构、随机值生成和角色比较，不访问数据库。

// generatePrefix 将四个密码学随机字节编码为八位十六进制查询前缀。
// 前缀是可公开的定位信息，完整密钥还必须包含独立生成的密钥部分。
func generatePrefix() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generateSecret 将三十二个密码学随机字节编码为六十四位十六进制密钥文本。
// 随机读取失败直接返回错误，调用方只把其摘要写入持久存储。
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// LooksLikeAPIKeyToken 只检查 lw_ 开头，供中间件选择认证方式。
// 此结果不代表格式或密钥有效，带此前缀的残缺令牌也会返回 true。
func LooksLikeAPIKeyToken(raw string) bool {
	return strings.HasPrefix(raw, apiKeyTokenPrefix)
}

// parseKeyToken 去掉固定开头后，以第一个下划线拆出非空前缀和密钥。
// 不裁剪空白，不验证十六进制、固定长度或密钥中其余下划线。
func parseKeyToken(token string) (prefix, secret string, ok bool) {
	if !strings.HasPrefix(token, apiKeyTokenPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(token, apiKeyTokenPrefix)
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// 角色顺序供 intersectRole 比较，数值越小权限越低。
var roleRank = map[string]int{
	RoleViewer: 1,
	RoleEditor: 2,
	RoleAdmin:  3,
}

// intersectRole 返回账号角色与密钥角色中权限较低的一项。
// 例如管理员持有只读密钥时只获得只读权限，编辑者持有管理员密钥时仍只有编辑权限。
// 任一角色不在排序表中均返回只读角色，本函数不为未知角色返回错误。
func intersectRole(userRole, keyRole string) string {
	uRank, uOK := roleRank[userRole]
	kRank, kOK := roleRank[keyRole]
	if !uOK || !kOK {
		return RoleViewer
	}
	if uRank < kRank {
		return userRole
	}
	return keyRole
}
