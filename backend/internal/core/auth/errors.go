// 本文件集中定义认证错误标识，调用方可用 errors.Is 区分失败原因。
// 错误文本属于程序返回内容，界面翻译与 HTTP 状态映射由上层完成。
package auth

import "errors"

// 用户类错误覆盖查找、唯一性、凭据和角色约束，定义处不执行状态检查。
var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists")
var ErrUserInvalidCredentials = errors.New("invalid credentials")
var ErrUserInvalidRole = errors.New("invalid role")
var ErrUserAdminCannotBeDeleted = errors.New("admin user cannot be deleted; change role before deletion")
var ErrLastAdminCannotBeDemoted = errors.New("cannot remove admin role from the last admin user")
var ErrInvalidToken = errors.New("invalid token")
var ErrSessionManagerNotWired = errors.New("session manager: resolveUser not configured (NewAuthService wires this; a SessionManager built directly must set it before use)")
var ErrUserAccountLocked = errors.New("account temporarily locked due to too many failed login attempts")
var ErrRemoteUserEmailConflict = errors.New("remote user auto-create: asserted email belongs to a different existing user")
var ErrPasswordTooShort = errors.New("password is too short")
var ErrEditorLimitReached = errors.New("editor limit reached for this plan")

// API 密钥错误分别区分查找失败、无效、撤销、到期和前缀冲突。
var ErrAPIKeyNotFound = errors.New("api key not found")
var ErrAPIKeyInvalid = errors.New("invalid api key")
var ErrAPIKeyRevoked = errors.New("api key has been revoked")
var ErrAPIKeyExpired = errors.New("api key has expired")
var ErrAPIKeyPrefixCollision = errors.New("api key prefix collision")

// 邮件令牌错误向上层表达无效令牌、邮件未启用和邀请已接受等结果。
var ErrEmailTokenInvalid = errors.New("invalid or expired token")
var ErrEmailDisabled = errors.New("email is not configured")
var ErrInviteAlreadyAccepted = errors.New("invite has already been accepted")

// ErrCodeAuthUserStoreUnavailable 标识用户存储暂停期间的本地化不可用错误。
const ErrCodeAuthUserStoreUnavailable = "auth_user_store_unavailable"
