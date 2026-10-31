package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/favorites"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// ErrAuthUnavailable 表示登录、登出或刷新入口缺少鉴权服务。
var ErrAuthUnavailable = errors.New("authentication service unavailable")

// emailRegex 只检查本地部分、一个 @ 和域名字符范围，不验证 DNS、收件能力或完整邮件标准。
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+$`)

// LoginInput 原样携带用户名或邮箱以及密码，验证由鉴权服务执行。
type LoginInput struct {
	Identifier string
	Password   string
}

// LoginOutput 返回访问与刷新令牌、公开用户信息和有效期，供路由写 Cookie。
type LoginOutput struct {
	Token *coreauth.AuthToken
}

// LoginUseCase 在鉴权服务调用后按结果登记登录及会话指标，不直接设置 Cookie。
type LoginUseCase struct {
	auth    *coreauth.AuthService
	metrics *httpmetrics.HTTPMetrics
}

// NewLoginUseCase 保存鉴权与可选指标，指标方法支持空接收者。
func NewLoginUseCase(a *coreauth.AuthService, metrics *httpmetrics.HTTPMetrics) *LoginUseCase {
	return &LoginUseCase{auth: a, metrics: metrics}
}

// Execute 将锁定、凭据错误及其他失败分别计数，成功时登记会话签发。
// 服务为空时直接返回，不登记尝试；请求上下文不传入底层鉴权操作。
func (uc *LoginUseCase) Execute(_ context.Context, in LoginInput) (*LoginOutput, error) {
	if uc.auth == nil {
		return nil, ErrAuthUnavailable
	}
	token, err := uc.auth.Login(in.Identifier, in.Password)
	if err != nil {
		switch {
		case errors.Is(err, coreauth.ErrUserAccountLocked):
			uc.metrics.IncAuthLoginAttempt("locked")
		case errors.Is(err, coreauth.ErrUserInvalidCredentials):
			uc.metrics.IncAuthLoginAttempt("invalid_credentials")
		default:
			uc.metrics.IncAuthLoginAttempt("error")
		}
		return nil, err
	}
	uc.metrics.IncAuthLoginAttempt("success")
	uc.metrics.IncAuthSession("issued")
	return &LoginOutput{Token: token}, nil
}

// LogoutInput 指定要撤销的刷新令牌，不凭访问令牌查找会话。
type LogoutInput struct{ RefreshToken string }

// LogoutUseCase 撤销持久化刷新会话，Cookie 清理由 HTTP 层另行执行。
type LogoutUseCase struct {
	auth    *coreauth.AuthService
	metrics *httpmetrics.HTTPMetrics
}

// NewLogoutUseCase 保存撤销入口及会话事件指标。
func NewLogoutUseCase(a *coreauth.AuthService, metrics *httpmetrics.HTTPMetrics) *LogoutUseCase {
	return &LogoutUseCase{auth: a, metrics: metrics}
}

// Execute 仅在撤销调用成功后累计 revoked 事件，失败原样返回。
func (uc *LogoutUseCase) Execute(_ context.Context, in LogoutInput) error {
	if uc.auth == nil {
		return ErrAuthUnavailable
	}
	if err := uc.auth.RevokeRefreshToken(in.RefreshToken); err != nil {
		return err
	}
	uc.metrics.IncAuthSession("revoked")
	return nil
}

// RefreshTokenInput 提供当前刷新令牌，轮换规则由鉴权服务统一执行。
type RefreshTokenInput struct{ RefreshToken string }

// RefreshTokenOutput 携带服务签发的令牌对，供路由替换浏览器 Cookie。
type RefreshTokenOutput struct {
	Token *coreauth.AuthToken
}

// RefreshTokenUseCase 调用刷新入口，并在成功时登记 refreshed 会话事件。
type RefreshTokenUseCase struct {
	auth    *coreauth.AuthService
	metrics *httpmetrics.HTTPMetrics
}

// NewRefreshTokenUseCase 保存鉴权与指标依赖。
func NewRefreshTokenUseCase(a *coreauth.AuthService, metrics *httpmetrics.HTTPMetrics) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{auth: a, metrics: metrics}
}

// Execute 拒绝空服务，刷新错误不登记成功事件，也不在该层修改 Cookie。
func (uc *RefreshTokenUseCase) Execute(_ context.Context, in RefreshTokenInput) (*RefreshTokenOutput, error) {
	if uc.auth == nil {
		return nil, ErrAuthUnavailable
	}
	token, err := uc.auth.RefreshToken(in.RefreshToken)
	if err != nil {
		return nil, err
	}
	uc.metrics.IncAuthSession("refreshed")
	return &RefreshTokenOutput{Token: token}, nil
}

// CreateUserInput 提供账号身份、明文初始密码与角色，不携带创建者权限信息。
type CreateUserInput struct {
	Username string
	Email    string
	Password string
	Role     string
}

// CreateUserOutput 只返回公开用户形态，不暴露密码摘要。
type CreateUserOutput struct {
	User *coreauth.PublicUser
}

// CreateUserUseCase 汇总输入错误，创建账号后刷新共享作者名称缓存。
type CreateUserUseCase struct {
	// user 每次执行时获取当前用户服务，跟随恢复流程中的存储替换。
	user     func() *coreauth.UserService
	resolver *coreauth.UserResolver
	log      *slog.Logger
}

// NewCreateUserUseCase 保存动态服务入口与缓存、日志，不预先读取账号。
func NewCreateUserUseCase(u func() *coreauth.UserService, r *coreauth.UserResolver, log *slog.Logger) *CreateUserUseCase {
	return &CreateUserUseCase{user: u, resolver: r, log: log}
}

// Execute 检查空文本、邮箱字符形式、密码字节长度与角色，再交给用户服务处理唯一性等约束。
// 不在此处裁去输入空白；缓存刷新失败只记警告，不撤销已创建账号。
func (uc *CreateUserUseCase) Execute(_ context.Context, in CreateUserInput) (*CreateUserOutput, error) {
	ve := sharederrors.NewValidationErrors()
	if in.Username == "" {
		ve.Add("username", "Username must not be empty")
	}
	if in.Email == "" {
		ve.Add("email", "Email must not be empty")
	} else if !emailRegex.MatchString(in.Email) {
		ve.Add("email", "Email is not valid")
	}
	if in.Password == "" {
		ve.Add("password", "Password must not be empty")
	} else if len(in.Password) < coreauth.MinPasswordLength {
		ve.Add("password", fmt.Sprintf("Password must be at least %d characters long", coreauth.MinPasswordLength))
	}
	if !coreauth.IsValidRole(in.Role) {
		ve.Add("role", "Invalid role")
	}
	if ve.HasErrors() {
		return nil, ve
	}

	user, err := uc.user().CreateUser(in.Username, in.Email, in.Password, in.Role)
	if err != nil {
		return nil, err
	}
	if err := uc.resolver.Reload(); err != nil {
		uc.log.Warn("failed to reload user resolver cache", "error", err)
	}
	return &CreateUserOutput{User: user.ToPublicUser()}, nil
}

// UpdateUserInput 同时携带目标字段和请求者身份，角色写入规则依据请求者管理员标记。
type UpdateUserInput struct {
	ID               string
	Username         string
	Email            string
	Password         string
	Role             string
	RequesterID      string
	RequesterIsAdmin bool
}

// UpdateUserOutput 返回成功更新后的公开账号。
type UpdateUserOutput struct {
	User *coreauth.PublicUser
}

// UpdateUserUseCase 负责输入校验、角色保留、作者缓存刷新及可选刷新会话撤销。
type UpdateUserUseCase struct {
	user     func() *coreauth.UserService
	resolver *coreauth.UserResolver
	auth     *coreauth.AuthService // 为空时跳过密码更新后的刷新会话撤销。
	log      *slog.Logger
}

// NewUpdateUserUseCase 保存动态用户服务及更新后的附带处理入口。
func NewUpdateUserUseCase(u func() *coreauth.UserService, r *coreauth.UserResolver, a *coreauth.AuthService, log *slog.Logger) *UpdateUserUseCase {
	return &UpdateUserUseCase{user: u, resolver: r, auth: a, log: log}
}

// Execute 不允许通过该入口为自己设置非空密码，自助改密必须通过验证旧密码的入口。
// 非管理员或未提供角色时读取并保留原角色；管理员提供的非空角色必须精确合法。
// 目标账号访问权限由路由检查，此用例不自行验证请求者是否有权修改目标用户。
func (uc *UpdateUserUseCase) Execute(_ context.Context, in UpdateUserInput) (*UpdateUserOutput, error) {
	ve := sharederrors.NewValidationErrors()
	if in.Username == "" {
		ve.Add("username", "Username must not be empty")
	}
	if in.Email == "" {
		ve.Add("email", "Email must not be empty")
	} else if !emailRegex.MatchString(in.Email) {
		ve.Add("email", "Email is not valid")
	}
	if in.Password != "" {
		// 当前账号改密不能利用管理员更新入口跳过旧密码验证。
		if in.ID == in.RequesterID {
			ve.Add("password", "Use the account settings to change your own password")
		} else if len(in.Password) < coreauth.MinPasswordLength {
			ve.Add("password", fmt.Sprintf("Password must be at least %d characters long", coreauth.MinPasswordLength))
		}
	}
	role := in.Role
	roleProvided := strings.TrimSpace(in.Role) != ""
	if in.RequesterIsAdmin && roleProvided && !coreauth.IsValidRole(in.Role) {
		ve.Add("role", "Invalid role")
	}
	if ve.HasErrors() {
		return nil, ve
	}

	if !in.RequesterIsAdmin || !roleProvided {
		existing, err := uc.user().GetUserByID(in.ID)
		if err != nil {
			return nil, err
		}
		role = existing.Role
	}

	user, err := uc.user().UpdateUser(in.ID, in.Username, in.Email, in.Password, role)
	if err != nil {
		return nil, err
	}
	if err := uc.resolver.Reload(); err != nil {
		uc.log.Warn("failed to reload user resolver cache", "error", err)
	}
	// 非空密码保存后尝试撤销目标用户的持久化刷新会话，失败只记警告。
	// 此调用不直接使已签发的访问 JWT 失效，也不回滚已经保存的密码。
	if in.Password != "" && uc.auth != nil {
		if err := uc.auth.RevokeAllUserSessions(in.ID); err != nil {
			uc.log.Warn("failed to revoke sessions after admin password change", "userID", in.ID, "error", err)
		}
	}
	return &UpdateUserOutput{User: user.ToPublicUser()}, nil
}

// ChangeOwnPasswordInput 提供当前用户、新旧密码与用来保留当前刷新会话的令牌。
type ChangeOwnPasswordInput struct {
	UserID              string
	OldPassword         string
	NewPassword         string
	CurrentRefreshToken string
}

// ChangeOwnPasswordUseCase 先验证新密码长度及旧密码，再写密码并尝试撤销其他刷新会话。
type ChangeOwnPasswordUseCase struct {
	user func() *coreauth.UserService
	auth *coreauth.AuthService // 为空时仅写密码，不执行刷新会话撤销。
	log  *slog.Logger
}

// NewChangeOwnPasswordUseCase 保存动态用户服务和可选会话撤销服务。
func NewChangeOwnPasswordUseCase(u func() *coreauth.UserService, a *coreauth.AuthService, log *slog.Logger) *ChangeOwnPasswordUseCase {
	return &ChangeOwnPasswordUseCase{user: u, auth: a, log: log}
}

// Execute 即使新密码验证已有错误也检查旧密码，将旧密码校验的任意错误归为字段错误。
// 验证通过后用户服务再次验证旧密码并执行写入，会话撤销失败不改变改密成功结果。
func (uc *ChangeOwnPasswordUseCase) Execute(_ context.Context, in ChangeOwnPasswordInput) error {
	ve := sharederrors.NewValidationErrors()
	if in.NewPassword == "" {
		ve.Add("newPassword", "New password must not be empty")
	} else if len(in.NewPassword) < coreauth.MinPasswordLength {
		ve.Add("newPassword", fmt.Sprintf("New password must be at least %d characters long", coreauth.MinPasswordLength))
	}
	if _, err := uc.user().DoesIDAndPasswordMatch(in.UserID, in.OldPassword); err != nil {
		ve.Add("oldPassword", "Old password is incorrect")
	}
	if ve.HasErrors() {
		return ve
	}
	if err := uc.user().ChangeOwnPassword(in.UserID, in.OldPassword, in.NewPassword); err != nil {
		return err
	}
	// 有效的当前刷新令牌用于保留对应持久化会话，空或无效令牌遵循服务的全撤销回退。
	// 这里只操作刷新会话记录，不直接撤销访问 JWT。
	if uc.auth != nil {
		if err := uc.auth.RevokeAllUserSessionsExceptCurrent(in.UserID, in.CurrentRefreshToken); err != nil {
			uc.log.Warn("failed to revoke other sessions after password change", "userID", in.UserID, "error", err)
		}
	}
	return nil
}

// DeleteUserInput 指定要删除的用户标识，权限不由这个输入结构表达。
type DeleteUserInput struct{ ID string }

// DeleteUserUseCase 删除账号后依次刷新作者缓存、删除收藏及可选密钥记录。
type DeleteUserUseCase struct {
	user      func() *coreauth.UserService
	resolver  *coreauth.UserResolver
	favorites *favorites.FavoritesStore
	apiKeys   *coreauth.APIKeyService // 密钥功能关闭时为空，跳过密钥记录清理。
	log       *slog.Logger
}

// NewDeleteUserUseCase 保存账号删除后的关联清理依赖，不构造跨存储事务。
func NewDeleteUserUseCase(u func() *coreauth.UserService, r *coreauth.UserResolver, f *favorites.FavoritesStore, ak *coreauth.APIKeyService, log *slog.Logger) *DeleteUserUseCase {
	return &DeleteUserUseCase{user: u, resolver: r, favorites: f, apiKeys: ak, log: log}
}

// Execute 只有主账号删除错误会作为失败返回，后续缓存与关联数据清理错误均记录警告。
// 这些步骤不共享事务，因此后续失败不恢复账号，传入上下文也不参与取消。
func (uc *DeleteUserUseCase) Execute(_ context.Context, in DeleteUserInput) error {
	if err := uc.user().DeleteUser(in.ID); err != nil {
		return err
	}
	if err := uc.resolver.Reload(); err != nil {
		uc.log.Warn("failed to reload user resolver cache", "error", err)
	}
	if err := uc.favorites.DeleteAllForUser(in.ID); err != nil {
		uc.log.Warn("failed to delete favorites for deleted user", "userID", in.ID, "error", err)
	}
	if uc.apiKeys != nil {
		if err := uc.apiKeys.DeleteAllForUser(in.ID); err != nil {
			uc.log.Warn("failed to delete api keys for deleted user", "userID", in.ID, "error", err)
		}
	}
	return nil
}

// GetUsersOutput 保存公开用户列表，不将存储中的密码摘要传给路由。
type GetUsersOutput struct {
	Users []*coreauth.PublicUser
}

// GetUsersUseCase 每次查询动态取得当前用户服务。
type GetUsersUseCase struct {
	user func() *coreauth.UserService
}

// NewGetUsersUseCase 保存服务获取函数，使查询跟随恢复后的存储。
func NewGetUsersUseCase(u func() *coreauth.UserService) *GetUsersUseCase {
	return &GetUsersUseCase{user: u}
}

// Execute 读取全部账号后逐项转为公开结构，顺序沿用服务结果。
func (uc *GetUsersUseCase) Execute(_ context.Context) (*GetUsersOutput, error) {
	users, err := uc.user().GetUsers()
	if err != nil {
		return nil, err
	}
	pub := make([]*coreauth.PublicUser, len(users))
	for i, u := range users {
		pub[i] = u.ToPublicUser()
	}
	return &GetUsersOutput{Users: pub}, nil
}

// GetUserByIDInput 指定待查询的稳定用户标识。
type GetUserByIDInput struct{ ID string }

// GetUserByIDOutput 返回单个公开账号，不暴露密码字段。
type GetUserByIDOutput struct {
	User *coreauth.PublicUser
}

// GetUserByIDUseCase 将标识查询交给当前用户服务。
type GetUserByIDUseCase struct {
	user func() *coreauth.UserService
}

// NewGetUserByIDUseCase 保存动态服务入口，不缓存目标用户。
func NewGetUserByIDUseCase(u func() *coreauth.UserService) *GetUserByIDUseCase {
	return &GetUserByIDUseCase{user: u}
}

// Execute 原样返回查找错误，成功时转换为公开结构，不检查传入上下文。
func (uc *GetUserByIDUseCase) Execute(_ context.Context, in GetUserByIDInput) (*GetUserByIDOutput, error) {
	user, err := uc.user().GetUserByID(in.ID)
	if err != nil {
		return nil, err
	}
	return &GetUserByIDOutput{User: user.ToPublicUser()}, nil
}

// RequestPasswordResetInput 携带用户提交的标识，不向调用方返回是否存在对应账号。
type RequestPasswordResetInput struct {
	Identifier string
}

// RequestPasswordResetUseCase 将实际账号查询与发送交给后台邮件任务。
type RequestPasswordResetUseCase struct {
	// emailTokens 每次执行时获取，未配置邮件服务时返回空值。
	emailTokens func() *coreauth.EmailTokenService
}

// NewRequestPasswordResetUseCase 保存动态邮件服务获取入口。
func NewRequestPasswordResetUseCase(e func() *coreauth.EmailTokenService) *RequestPasswordResetUseCase {
	return &RequestPasswordResetUseCase{emailTokens: e}
}

// Execute 只在邮件服务缺失时返回错误，否则提交后台任务后立即返回成功。
// 不等待账号查询、令牌签发或邮件发送，也不向当前请求报告这些后台结果。
// 该入口不通过请求结果区分账号是否存在，任务关闭等待由邮件服务负责。
func (uc *RequestPasswordResetUseCase) Execute(_ context.Context, in RequestPasswordResetInput) error {
	tokens := uc.emailTokens()
	if tokens == nil {
		return coreauth.ErrEmailDisabled
	}
	tokens.RequestPasswordResetAsync(in.Identifier)
	return nil
}

// ConfirmPasswordResetInput 提供一次性邮件令牌与待写入的新密码。
type ConfirmPasswordResetInput struct {
	Token       string
	NewPassword string
}

// ConfirmPasswordResetOutput 返回公开用户，不包含登录会话或 Cookie。
type ConfirmPasswordResetOutput struct {
	User *coreauth.PublicUser
}

// ConfirmPasswordResetUseCase 检查密码后委托邮件令牌服务完成令牌消费与密码写入。
type ConfirmPasswordResetUseCase struct {
	emailTokens func() *coreauth.EmailTokenService
}

// NewConfirmPasswordResetUseCase 保存动态邮件服务入口，不在构造时验证令牌。
func NewConfirmPasswordResetUseCase(e func() *coreauth.EmailTokenService) *ConfirmPasswordResetUseCase {
	return &ConfirmPasswordResetUseCase{emailTokens: e}
}

// Execute 要求非空且达到最短字节长度的新密码，成功后不签发会话。
// 底层流程消费令牌、写密码并尝试撤销刷新会话，撤销失败的处理由邮件服务决定。
// 这里不能据成功输出断言全部访问 JWT 已经失效。
func (uc *ConfirmPasswordResetUseCase) Execute(_ context.Context, in ConfirmPasswordResetInput) (*ConfirmPasswordResetOutput, error) {
	tokens := uc.emailTokens()
	if tokens == nil {
		return nil, coreauth.ErrEmailDisabled
	}

	ve := sharederrors.NewValidationErrors()
	if in.NewPassword == "" {
		ve.Add("newPassword", "New password must not be empty")
	} else if len(in.NewPassword) < coreauth.MinPasswordLength {
		ve.Add("newPassword", fmt.Sprintf("New password must be at least %d characters long", coreauth.MinPasswordLength))
	}
	if ve.HasErrors() {
		return nil, ve
	}

	user, err := tokens.ConfirmPasswordReset(in.Token, in.NewPassword)
	if err != nil {
		return nil, err
	}
	return &ConfirmPasswordResetOutput{User: user.ToPublicUser()}, nil
}

// InviteUserInput 提供待邀请账号身份与角色，不从管理员接收初始密码。
type InviteUserInput struct {
	Username string
	Email    string
	Role     string
}

// InviteUserOutput 将账号创建结果与邮件发送结果分开表达，发送失败时账号仍存在。
type InviteUserOutput struct {
	User      *coreauth.PublicUser
	EmailSent bool
}

// InviteUserUseCase 创建待设密码账号、刷新作者缓存，再同步尝试签发邀请邮件。
type InviteUserUseCase struct {
	user        func() *coreauth.UserService
	emailTokens func() *coreauth.EmailTokenService
	resolver    *coreauth.UserResolver
	log         *slog.Logger
}

// NewInviteUserUseCase 保存用户及邮件动态入口，复用现有缓存与日志。
func NewInviteUserUseCase(u func() *coreauth.UserService, e func() *coreauth.EmailTokenService, r *coreauth.UserResolver, log *slog.Logger) *InviteUserUseCase {
	return &InviteUserUseCase{user: u, emailTokens: e, resolver: r, log: log}
}

// Execute 在创建前要求邮件服务可用，并检查用户名、邮箱字符形式与角色。
// 账号创建失败返回错误；创建后缓存刷新或邮件发送失败只记录警告，不回滚账号。
// 发送结果通过 EmailSent 表达，请求上下文仅传给邀请邮件签发入口。
func (uc *InviteUserUseCase) Execute(ctx context.Context, in InviteUserInput) (*InviteUserOutput, error) {
	tokens := uc.emailTokens()
	if tokens == nil {
		return nil, coreauth.ErrEmailDisabled
	}

	ve := sharederrors.NewValidationErrors()
	if in.Username == "" {
		ve.Add("username", "Username must not be empty")
	}
	if in.Email == "" {
		ve.Add("email", "Email must not be empty")
	} else if !emailRegex.MatchString(in.Email) {
		ve.Add("email", "Email is not valid")
	}
	if !coreauth.IsValidRole(in.Role) {
		ve.Add("role", "Invalid role")
	}
	if ve.HasErrors() {
		return nil, ve
	}

	user, err := uc.user().InviteUser(in.Username, in.Email, in.Role)
	if err != nil {
		return nil, err
	}
	if err := uc.resolver.Reload(); err != nil {
		uc.log.Warn("failed to reload user resolver cache", "error", err)
	}

	emailSent := true
	if err := tokens.IssueInvite(ctx, user); err != nil {
		uc.log.Warn("failed to send invite email", "userID", user.ID, "error", err)
		emailSent = false
	}

	return &InviteUserOutput{User: user.ToPublicUser(), EmailSent: emailSent}, nil
}

// ResendInviteInput 指定已有待设密码账号，不按邮箱直接创建账号。
type ResendInviteInput struct {
	UserID string
}

// ResendInviteUseCase 检查账号邀请状态后再次调用邮件签发。
type ResendInviteUseCase struct {
	user        func() *coreauth.UserService
	emailTokens func() *coreauth.EmailTokenService
}

// NewResendInviteUseCase 保存动态用户与邮件服务入口。
func NewResendInviteUseCase(u func() *coreauth.UserService, e func() *coreauth.EmailTokenService) *ResendInviteUseCase {
	return &ResendInviteUseCase{user: u, emailTokens: e}
}

// Execute 拒绝未配置邮件、找不到账号或已经不再需要设置密码的账号。
// 与创建邀请不同，此入口直接返回发送错误，成功没有附带账号输出。
func (uc *ResendInviteUseCase) Execute(ctx context.Context, in ResendInviteInput) error {
	tokens := uc.emailTokens()
	if tokens == nil {
		return coreauth.ErrEmailDisabled
	}

	user, err := uc.user().GetUserByID(in.UserID)
	if err != nil {
		return err
	}
	if !user.MustSetPassword {
		return coreauth.ErrInviteAlreadyAccepted
	}

	return tokens.IssueInvite(ctx, user)
}

// ConfirmInviteInput 提供邀请令牌与用户选择的新密码。
type ConfirmInviteInput struct {
	Token       string
	NewPassword string
}

// ConfirmInviteOutput 返回邀请确认后签发的会话，供 HTTP 层直接写入登录 Cookie。
type ConfirmInviteOutput struct {
	Token *coreauth.AuthToken
}

// ConfirmInviteUseCase 顺序完成邀请确认和会话签发，两个动作不共享这里创建的事务。
type ConfirmInviteUseCase struct {
	emailTokens func() *coreauth.EmailTokenService
	auth        *coreauth.AuthService
}

// NewConfirmInviteUseCase 保存邮件动态入口与会话签发服务，调用方保证后者可用。
func NewConfirmInviteUseCase(e func() *coreauth.EmailTokenService, a *coreauth.AuthService) *ConfirmInviteUseCase {
	return &ConfirmInviteUseCase{emailTokens: e, auth: a}
}

// Execute 检查密码长度后通过邮件服务消费令牌并设置密码，再按用户标识签发会话。
// 会话签发失败会返回错误，但不会撤销已经完成的邀请确认或恢复邮件令牌。
// 请求上下文未传入确认与签发操作。
func (uc *ConfirmInviteUseCase) Execute(_ context.Context, in ConfirmInviteInput) (*ConfirmInviteOutput, error) {
	tokens := uc.emailTokens()
	if tokens == nil {
		return nil, coreauth.ErrEmailDisabled
	}

	ve := sharederrors.NewValidationErrors()
	if in.NewPassword == "" {
		ve.Add("newPassword", "New password must not be empty")
	} else if len(in.NewPassword) < coreauth.MinPasswordLength {
		ve.Add("newPassword", fmt.Sprintf("New password must be at least %d characters long", coreauth.MinPasswordLength))
	}
	if ve.HasErrors() {
		return nil, ve
	}

	user, err := tokens.ConfirmInvite(in.Token, in.NewPassword)
	if err != nil {
		return nil, err
	}

	token, err := uc.auth.IssueSessionForUser(user.ID)
	if err != nil {
		return nil, err
	}
	return &ConfirmInviteOutput{Token: token}, nil
}
