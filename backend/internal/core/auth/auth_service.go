package auth

import (
	"errors"
	"log/slog"
	"sync"

	"golang.org/x/crypto/bcrypt"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// AuthService 组织账号密码验证、进程内尝试限制和访问刷新令牌签发。
// 用户服务可在恢复流程中替换，会话管理器通过动态解析回调读取当前服务。
type AuthService struct {
	// mu 只保护用户服务指针的读取与替换，不包住完整账号查询或密码比较。
	mu          sync.RWMutex
	userService *UserService
	sessions    *SessionManager
	attempts    *loginAttemptTracker
	dummyHash   []byte
	log         *slog.Logger
}

// users 在读锁内取得当前用户服务指针，返回后释放锁。
// 后续操作继续使用该指针，不会因中途替换而自动转移到另一用户服务。
func (a *AuthService) users() *UserService {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.userService
}

// UserService 返回调用时绑定的账号服务，不复制其内部存储状态。
func (a *AuthService) UserService() *UserService {
	return a.users()
}

// ReplaceUserStore 先建立指定目录的用户存储，再在写锁内替换用户服务。
// 替换后尝试关闭旧服务，关闭失败只记录告警，不撤销已经完成的指针替换。
func (a *AuthService) ReplaceUserStore(storageDir string) error {
	newStore, err := NewUserStore(storageDir)
	if err != nil {
		return err
	}
	newUserService := NewUserService(newStore)

	a.mu.Lock()
	old := a.userService
	// 编辑人数限制从旧服务复制，不从恢复后的用户数据重新推导。
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

// PauseUserStoreForSwap 暂停当前服务的账号存储访问，错误由存储暂停入口返回。
func (a *AuthService) PauseUserStoreForSwap() error {
	return a.users().suspendStore()
}

// InvalidateAllSessions 删除全部持久化刷新会话，不在此更换签名密钥或认证代际。
// 访问令牌本身不在会话表登记，其失效还遵循会话管理器的代际和账号验证规则。
func (a *AuthService) InvalidateAllSessions() error {
	return a.sessions.sessionStore.DeleteAllSessions()
}

// NewAuthService 建立尝试计数器和日志器，并为会话管理器绑定动态用户解析回调。
// 调用方须提供会话管理器，构造直接给该对象设置回调。
func NewAuthService(userService *UserService, sessions *SessionManager) *AuthService {
	// 预先生成占位密码哈希，账号查询失败时仍执行一次密码比较。
	// 这使失败路径包含哈希计算，不声明整个请求的执行时间完全相同。
	dummyHash, _ := bcrypt.GenerateFromPassword([]byte("leafdoc-dummy-password"), bcrypt.DefaultCost)
	a := &AuthService{
		userService: userService,
		sessions:    sessions,
		attempts:    newLoginAttemptTracker(),
		dummyHash:   dummyHash,
		log:         slog.Default().With("component", "AuthService"),
	}
	// 每次回调都重新读取当前用户服务，闭包不固定捕获构造时的用户服务对象。
	sessions.resolveUser = func(id string) (*User, error) {
		return a.users().GetUserByID(id)
	}
	return a
}

// Close 依次关闭会话管理器和当前用户服务，收集两者错误而不因首个失败跳过后者。
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

// AuthToken 返回访问令牌、刷新令牌、访问到期秒数和不含密码的公开用户视图。
type AuthToken struct {
	Token                string      `json:"token"`
	RefreshToken         string      `json:"refresh_token"`
	AccessTokenExpiresAt int64       `json:"accessTokenExpiresAt"`
	User                 *PublicUser `json:"user"`
}

// Login 先按标识读取用户，再登记尝试并比较密码，成功后清除尝试状态并签发会话。
// 账号查询的所有错误统一转为凭据无效，不把底层存储原因直接返回此调用方。
// 第五次登记仍可比较密码，已有锁定状态才在比较之前拒绝请求。
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

// IssueSessionForUser 按标识读取用户后直接签发，不在此校验密码或登记登录尝试。
// 调用方负责在进入此入口前完成需要的身份验证流程。
func (a *AuthService) IssueSessionForUser(userID string) (*AuthToken, error) {
	user, err := a.users().GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return a.sessions.IssueSession(user)
}

// errUserStoreUnavailable 构造账号存储暂停期间使用的结构化错误。
// 返回值包含稳定错误码和翻译模板，不在此恢复连接或等待恢复操作结束。
func errUserStoreUnavailable() error {
	return sharederrors.NewLocalizedError(
		ErrCodeAuthUserStoreUnavailable,
		"The server is restoring from a backup — please try again in a moment",
		"user store is suspended for an in-progress restore",
		nil,
	)
}

// RefreshToken 将刷新令牌校验与轮换交给会话管理器。
func (a *AuthService) RefreshToken(refreshToken string) (*AuthToken, error) {
	return a.sessions.RefreshToken(refreshToken)
}

// RevokeRefreshToken 通过令牌中的会话标识撤销单个刷新会话。
func (a *AuthService) RevokeRefreshToken(tokenString string) error {
	return a.sessions.RevokeRefreshToken(tokenString)
}

// RevokeAllUserSessions 撤销用户的持久化刷新会话，不在此重新签发访问令牌。
func (a *AuthService) RevokeAllUserSessions(userID string) error {
	return a.sessions.RevokeAllUserSessions(userID)
}

// RevokeAllUserSessionsExceptCurrent 委托会话管理器解析要保留的标识并撤销其余记录。
func (a *AuthService) RevokeAllUserSessionsExceptCurrent(userID, currentRefreshToken string) error {
	return a.sessions.RevokeAllUserSessionsExceptCurrent(userID, currentRefreshToken)
}

// ValidateToken 校验访问令牌并重新读取当前用户，不直接信任令牌内保存的角色。
func (a *AuthService) ValidateToken(tokenString string) (*User, error) {
	return a.sessions.ValidateToken(tokenString)
}
