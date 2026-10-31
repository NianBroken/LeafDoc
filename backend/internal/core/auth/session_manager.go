// 本文件负责 JWT 与刷新会话记录之间的配合，访问令牌只在请求时解析。
// 刷新令牌既要通过签名与声明检查，也要对应数据库中尚未撤销且未到期的记录。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionManager 签发带认证代际的 JWT，并用数据库记录管理刷新令牌的有效与撤销状态。
// 访问令牌不保存会话行，验证时检查签名、类型、代际并重新读取用户。
type SessionManager struct {
	sessionStore         *SessionStore // 保存刷新记录与所有令牌共用的认证代际。
	secretKey            []byte        // 从构造参数复制为字节切片，签发与解析使用同一密钥。
	accessTokenLifetime  time.Duration
	refreshTokenLifetime time.Duration
	log                  *slog.Logger

	// resolveUser 每次刷新或验证访问令牌时按主体标识读取用户，不在管理器中缓存结果。
	// NewAuthService 负责绑定该回调，使用户存储替换后继续从当前账号服务读取。
	resolveUser func(id string) (*User, error)

	// now 默认读取当前时间，测试可替换。JWT 解析器的时钟不在这里绑定。
	now func() time.Time
}

// NewSessionManager 保存存储、签名密钥和两种令牌寿命，密钥少于 32 字节时记录告警。
// 构造不拒绝较短密钥，也不在此绑定用户解析器，后续需由认证服务完成绑定。
func NewSessionManager(sessionStore *SessionStore, secret string, accessTokenTimeout, refreshTokenTimeout time.Duration) *SessionManager {
	log := slog.Default().With("component", "SessionManager")
	if len(secret) < 32 {
		log.Warn("JWT secret is too short; a minimum of 32 characters is strongly recommended", "length", len(secret))
	}
	return &SessionManager{
		sessionStore:         sessionStore,
		secretKey:            []byte(secret),
		accessTokenLifetime:  accessTokenTimeout,
		refreshTokenLifetime: refreshTokenTimeout,
		now:                  time.Now,
		log:                  log,
	}
}

// Close 关闭存在的会话存储，未配置存储时直接成功。
func (s *SessionManager) Close() error {
	if s.sessionStore == nil {
		return nil
	}
	return s.sessionStore.Close()
}

// IssueSession 为调用方已经确认身份的用户签发访问与刷新令牌，只保存刷新会话行。
// 任一令牌生成或刷新行插入失败都返回错误，不向调用方交付部分令牌。
func (s *SessionManager) IssueSession(user *User) (*AuthToken, error) {
	accessToken, _, accessTokenExpiresAt, err := s.generateToken(user, s.accessTokenLifetime, "access")
	if err != nil {
		return nil, err
	}

	refreshToken, refreshJTI, refreshExpiresAt, err := s.generateToken(user, s.refreshTokenLifetime, "refresh")
	if err != nil {
		return nil, err
	}

	// 持久化到期秒直接复用刷新令牌的 exp，不再次计算当前时间加寿命。
	// JWT 与数据库由此保存同一个到期数值，各自的有效性边界仍按对应验证规则处理。
	if err := s.sessionStore.CreateSession(
		refreshJTI,
		user.ID,
		"refresh",
		time.Unix(refreshExpiresAt, 0),
	); err != nil {
		return nil, err
	}

	s.log.Info("session issued", "userID", user.ID)
	return &AuthToken{
		Token:                accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
		User:                 user.ToPublicUser(),
	}, nil
}

// RefreshToken 检查 JWT、认证代际、刷新类型和持久化会话，再重新读取当前用户。
// 新令牌和新刷新行全部成功后才撤销旧行，前面的失败不会主动撤销旧刷新会话。
// 撤销旧行失败只记录告警，仍交付新令牌；此时旧行继续按自身撤销和到期规则判断。
func (s *SessionManager) RefreshToken(refreshToken string) (*AuthToken, error) {
	claims, err := s.parseClaims(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if err := s.validateGeneration(claims); err != nil {
		return nil, err
	}

	typ, ok := claims["typ"].(string)
	if !ok || typ != "refresh" {
		return nil, ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return nil, ErrInvalidToken
	}

	// 会话读取故障与无效记录统一表现为令牌无效，不直接返回该处数据库错误。
	active, err := s.sessionStore.IsActive(jti, userID, "refresh", s.now())
	if err != nil || !active {
		return nil, ErrInvalidToken
	}

	if s.resolveUser == nil {
		return nil, ErrSessionManagerNotWired
	}
	// 角色与公开资料重新读取，令牌中旧的 role 和 email 不作为当前账号状态。
	user, err := s.resolveUser(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	user.Password = ""

	newAccessToken, _, accessTokenExpiresAt, err := s.generateToken(user, s.accessTokenLifetime, "access")
	if err != nil {
		return nil, err
	}

	newRefreshToken, newRefreshJTI, newRefreshExpiresAt, err := s.generateToken(user, s.refreshTokenLifetime, "refresh")
	if err != nil {
		return nil, err
	}

	// 新刷新行同样复用令牌内到期秒，避免独立时间计算产生数值差异。
	if err := s.sessionStore.CreateSession(
		newRefreshJTI,
		user.ID,
		"refresh",
		time.Unix(newRefreshExpiresAt, 0),
	); err != nil {
		return nil, err
	}

	// 此处与上方插入分别调用存储，不由本方法包在同一个数据库事务中。
	if err := s.sessionStore.RevokeSession(jti); err != nil {
		s.log.Warn("failed to revoke used refresh token session", "error", err)
	}

	s.log.Info("session refreshed", "userID", user.ID)
	return &AuthToken{
		Token:                newAccessToken,
		RefreshToken:         newRefreshToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
		User:                 user.ToPublicUser(),
	}, nil
}

// RevokeRefreshToken 解析 JWT 并要求刷新类型和非空 jti，然后按标识标记撤销。
// 此入口不校验认证代际，也不查询该会话是否属于声明的主体，主体只用于日志。
func (s *SessionManager) RevokeRefreshToken(tokenString string) error {
	claims, err := s.parseClaims(tokenString)
	if err != nil {
		return ErrInvalidToken
	}

	typ, ok := claims["typ"].(string)
	if !ok || typ != "refresh" {
		return ErrInvalidToken
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return ErrInvalidToken
	}

	userID, _ := claims["sub"].(string)

	err = s.sessionStore.RevokeSession(jti)
	if err == nil {
		s.log.Info("session revoked", "userID", userID)
	}
	return err
}

// RevokeAllUserSessions 委托存储撤销用户的全部刷新会话记录。
func (s *SessionManager) RevokeAllUserSessions(userID string) error {
	return s.sessionStore.RevokeAllSessionsForUser(userID)
}

// RevokeAllUserSessionsExceptCurrent 从可解析 JWT 取得 jti，作为用户记录的排除标识。
// 此处不再检查令牌类型、主体或认证代际，解析失败或无字符串 jti 时使用空排除值。
func (s *SessionManager) RevokeAllUserSessionsExceptCurrent(userID, currentRefreshToken string) error {
	var exceptJTI string
	if claims, err := s.parseClaims(currentRefreshToken); err == nil {
		if jti, ok := claims["jti"].(string); ok {
			exceptJTI = jti
		}
	}
	return s.sessionStore.RevokeAllSessionsForUserExcept(userID, exceptJTI)
}

// ValidateToken 依次检查 JWT、认证代际、访问类型和字符串主体，再读取当前用户。
// 当前角色与账号状态来自用户服务，不使用令牌内 role 和 email 直接授权。
// 此流程不查询刷新会话撤销状态，撤销刷新记录不等同于即时撤销已有访问令牌。
func (s *SessionManager) ValidateToken(tokenString string) (*User, error) {
	claims, err := s.parseClaims(tokenString)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if err := s.validateGeneration(claims); err != nil {
		return nil, err
	}

	typ, ok := claims["typ"].(string)
	if !ok || typ != "access" {
		return nil, ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	if s.resolveUser == nil {
		return nil, ErrSessionManagerNotWired
	}
	return s.resolveUser(userID)
}

// generateJTI 读取 16 个加密随机字节并编码为 32 位十六进制令牌标识。
func generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// signClaims 使用 HS256 和当前密钥签名，不在此保存任何会话记录。
func (s *SessionManager) signClaims(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// parseClaims 使用 JWT 库默认校验流程解析令牌，密钥回调只接受 HMAC 方法类型。
// 此回调不单独限制为 HS256，实际签发入口固定使用 HS256。
// 解析失败、库判定无效或声明类型不符均返回统一令牌无效错误。
func (s *SessionManager) parseClaims(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// generateToken 生成随机标识并读取认证代际，再按给定寿命构造声明和签名。
// exp 与 iat 分别调用时钟，结果使用 Unix 秒；此方法只生成令牌，不创建会话行。
func (s *SessionManager) generateToken(user *User, duration time.Duration, typ string) (string, string, int64, error) {
	jti, err := generateJTI()
	if err != nil {
		return "", "", 0, err
	}
	if s.sessionStore == nil {
		return "", "", 0, ErrInvalidToken
	}
	generation, err := s.sessionStore.Generation()
	if err != nil {
		return "", "", 0, err
	}

	expiresAt := s.now().Add(duration).Unix()
	claims := jwt.MapClaims{
		"sub":             user.ID,
		"role":            user.Role,
		"email":           user.Email,
		"exp":             expiresAt,
		"iat":             s.now().Unix(),
		"typ":             typ,
		"jti":             jti, // 此令牌的随机标识，刷新令牌以它对应持久化会话行。
		"auth_generation": generation,
	}
	signed, err := s.signClaims(claims)
	if err != nil {
		return "", "", 0, err
	}
	return signed, jti, expiresAt, nil
}

// validateGeneration 要求声明中的非空代际与数据库当前值完全一致。
// 存储缺失、字段类型错误或值不符均拒绝，数据库读取错误直接向上传递。
// 此方法只比较值，不自行轮换代际，也不依据恢复发生时间推断令牌有效性。
func (s *SessionManager) validateGeneration(claims jwt.MapClaims) error {
	claimed, ok := claims["auth_generation"].(string)
	if !ok || claimed == "" || s.sessionStore == nil {
		return ErrInvalidToken
	}
	generation, err := s.sessionStore.Generation()
	if err != nil {
		return err
	}
	if claimed != generation {
		return ErrInvalidToken
	}
	return nil
}
