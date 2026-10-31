// 本文件实现用户创建、身份查询、角色变更、密码处理和管理员恢复。
// 持久化交给 UserStore，接口输入校验与会话撤销不在每个服务方法中统一执行。
package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/defaults"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAdminUsername = defaults.AdminUsername
	defaultAdminPassword = defaults.AdminPassword
	defaultAdminEmail    = defaults.AdminEmail

	// MinPasswordLength 为密码长度校验提供统一阈值，len 按 UTF-8 字节数计算。
	// 此文件的初始管理员和显式恢复密码直接检查该阈值，CreateUser、UpdateUser、
	// UpdatePassword 和 ChangeOwnPassword 不在方法内部执行此长度检查。
	MinPasswordLength = 8
)

// UserService 组合用户存储和日志器，编辑者配额属于该服务实例的配置。
type UserService struct {
	store *UserStore
	log   *slog.Logger
	// editorLimit 限制管理员与编辑者的合计人数，非正数表示不限制。
	// 创建账号和从未计数角色转入编辑角色时检查，配额查询与写入不构成一个事务。
	editorLimit int
}

// NewUserService 保存存储引用并派生带组件字段的默认日志器，不创建默认账号。
func NewUserService(store *UserStore) *UserService {
	return &UserService{
		store: store,
		log:   slog.Default().With("component", "UserService"),
	}
}

// SetEditorLimit 设置后续账号创建与角色升级使用的配额，不修改现有用户。
// 字段没有并发锁保护，应由装配流程在处理请求前配置。
func (s *UserService) SetEditorLimit(limit int) {
	s.editorLimit = limit
}

// checkEditorLimit 仅为管理员或编辑者统计当前人数，达到上限返回配额错误。
// 非正数上限或其他角色直接通过，数据库统计失败直接返回错误。
func (s *UserService) checkEditorLimit(role string) error {
	if s.editorLimit <= 0 || (role != RoleAdmin && role != RoleEditor) {
		return nil
	}
	count, err := s.store.CountEditorUsers()
	if err != nil {
		return err
	}
	if count >= s.editorLimit {
		return ErrEditorLimitReached
	}
	return nil
}

// InitDefaultAdmin 仅在没有管理员时创建初始账号，已有管理员直接成功。
// 参数裁去边缘空白后为空时采用集中默认值，密码检查通过后生成 bcrypt 摘要和用户 ID。
// 此路径直接写存储，不经过 CreateUser 的角色配额检查。
func (s *UserService) InitDefaultAdmin(username, email, newPassword string) error {
	if _, err := s.store.GetAdminUser(); err == nil {
		return nil
	} else if !errors.Is(err, ErrUserNotFound) {
		return err
	}
	// 调用方输入优先，空用户名、密码和邮箱分别回退为默认配置。
	username = defaultIfEmpty(username, defaultAdminUsername)
	newPassword = defaultIfEmpty(newPassword, defaultAdminPassword)
	if len(newPassword) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	id, err := shared.GenerateUniqueID()
	if err != nil {
		return err
	}
	user := &User{ID: id, Username: username, Email: defaultIfEmpty(email, defaultAdminEmail), Password: string(hashed), Role: RoleAdmin, MustChangePassword: username == "NianBroken" && newPassword == defaultAdminPassword}
	err = s.store.CreateUser(user)
	return err
}

// defaultIfEmpty 裁去输入边缘空白，仅在结果为空时返回原样 fallback。
func defaultIfEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

// CreateUser 检查用户名、邮箱冲突及角色配额后生成密码摘要和用户 ID，再保存账号。
// 本方法不检查字符串格式、裁去空白或密码最短长度；最终唯一性仍由存储约束执行。
func (s *UserService) CreateUser(username, email, password, role string) (*User, error) {
	// 查询成功表示用户名已占用，其他查询错误不会在这个预检查位置直接返回。
	_, err := s.store.GetUserByUsername(username)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	// 邮箱预检查采用相同规则，查询到用户时拒绝继续创建。
	_, err = s.store.GetUserByEmail(email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	// 角色必须精确匹配受支持值，再检查其是否占用编辑者配额。
	if !IsValidRole(role) {
		return nil, ErrUserInvalidRole
	}

	if err := s.checkEditorLimit(role); err != nil {
		return nil, err
	}

	// 使用 bcrypt 默认计算成本处理调用方原始密码，错误直接向上传递。
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 用户 ID 由共享随机 ID 生成器提供，与用户名和邮箱分离。
	id, err := shared.GenerateUniqueID()
	if err != nil {
		return nil, err
	}

	// 构造持久化用户对象，两个密码状态标志保留零值。
	user := &User{
		ID:       id,
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
		Role:     role,
	}

	// 存储成功后才记录创建日志，返回对象保留密码摘要供内部调用使用。
	err = s.store.CreateUser(user)
	if err != nil {
		return nil, err
	}

	s.log.Info("user created", "userID", user.ID, "role", user.Role)
	return user, nil
}

// mapUserLookupErr 仅保留带用户存储暂停错误码的本地化错误。
// 其余错误都转换成 ErrUserNotFound，包括非暂停类的数据库失败，不保留原错误。
func mapUserLookupErr(err error) error {
	if loc, ok := sharederrors.AsLocalizedError(err); ok && loc.Code == ErrCodeAuthUserStoreUnavailable {
		return err
	}
	return ErrUserNotFound
}

// GetUserByID 读取用户并按统一查找规则映射错误，成功结果保留内部密码字段。
func (s *UserService) GetUserByID(id string) (*User, error) {
	user, err := s.store.GetUserByID(id)
	if err != nil {
		return nil, mapUserLookupErr(err)
	}

	return user, nil
}

// UpdateUser 读取目标账号，检查唯一性、角色与配额，再保存身份字段及可选密码。
// 空密码保留原摘要，空白角色保留原角色，用户名和邮箱按传入值写入。
// 用户更新与待设密码标志清理分开执行，后一步失败不撤销已保存的用户字段。
func (s *UserService) UpdateUser(id, username, email, password, role string) (*User, error) {
	// 目标用户读取失败按查找规则映射，不继续执行更新。
	user, err := s.store.GetUserByID(id)
	if err != nil {
		return nil, mapUserLookupErr(err)
	}

	// 查到其他用户使用目标用户名时拒绝，同一用户本身不构成冲突。
	existingUser, err := s.store.GetUserByUsername(username)
	if err == nil && existingUser.ID != id {
		return nil, ErrUserAlreadyExists
	}

	// 邮箱同样只排除其他用户的占用，预检查查询错误不在这里直接返回。
	existingUser, err = s.store.GetUserByEmail(email)
	if err == nil && existingUser.ID != id {
		return nil, ErrUserAlreadyExists
	}

	if strings.TrimSpace(role) == "" {
		role = user.Role
	}

	// 空白角色沿用原值，但非空角色不先裁去空白，必须精确匹配角色表。
	if !IsValidRole(role) {
		return nil, ErrUserInvalidRole
	}

	// 管理员降级前统计管理员总数，唯一管理员不能失去管理员角色。
	if user.HasRole(RoleAdmin) && role != RoleAdmin {
		count, err := s.store.CountAdminUsers()
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, ErrLastAdminCannotBeDemoted
		}
	}

	// 原本已经是管理员或编辑者的账号不再占用额外名额。
	// 原角色未计入配额时才检查目标角色，目标仍为查看者时检查直接通过。
	wasCounted := user.Role == RoleAdmin || user.Role == RoleEditor
	if !wasCounted {
		if err := s.checkEditorLimit(role); err != nil {
			return nil, err
		}
	}

	// 保存旧角色用于日志分类，用户名、邮箱和角色赋值后再处理可选密码。
	oldRole := user.Role
	user.Username = username
	user.Email = email
	user.Role = role

	if password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashedPassword)
	}

	// 先保存用户字段，再处理邀请状态，两个存储调用没有共同事务。
	err = s.store.UpdateUser(user)
	if err != nil {
		return nil, err
	}

	// 传入非空密码且账号待设密码时，单独清理邀请状态。
	// 清理成功后同步返回对象的标志，失败则返回错误但已写入的密码仍保留。
	if password != "" && user.MustSetPassword {
		if err := s.store.SetMustSetPassword(user.ID, false); err != nil {
			return nil, err
		}
		user.MustSetPassword = false
		s.log.Info("invite completed via admin password change", "userID", user.ID)
	}

	if oldRole != user.Role {
		s.log.Info("user role changed", "userID", user.ID, "oldRole", oldRole, "newRole", user.Role)
	} else {
		s.log.Info("user updated", "userID", user.ID)
	}
	return user, nil
}

// UpdatePassword 确认用户存在后生成新摘要并转交存储，不验证旧密码或最短长度。
// 本方法不撤销会话，调用方负责业务层面的令牌清理。
func (s *UserService) UpdatePassword(id string, newpassword string) error {
	// 这里直接返回存储查找错误，不调用 mapUserLookupErr。
	_, err := s.store.GetUserByID(id)
	if err != nil {
		return err
	}

	// 按 bcrypt 默认成本处理新密码，输入内容不裁去空白。
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newpassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 保存摘要及存储层负责的密码状态，不将明文写入用户表。
	err = s.store.UpdatePassword(id, string(hashedPassword))
	if err != nil {
		return err
	}

	return nil
}

// DoesIDAndPasswordMatch 按 ID 查找并比较密码，查找失败按账号查找规则映射，
// 摘要比较失败返回凭据无效。此入口不登记登录尝试次数，也不清空成功结果的密码摘要。
func (s *UserService) DoesIDAndPasswordMatch(id, password string) (*User, error) {
	user, err := s.store.GetUserByID(id)
	if err != nil {
		return nil, mapUserLookupErr(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrUserInvalidCredentials
	}

	return user, nil
}

// DeleteUser 拒绝直接删除任何管理员，其他角色交给存储删除并记录日志。
// 收藏、会话和 API 密钥的业务清理由调用方编排，不在此方法中执行。
func (s *UserService) DeleteUser(id string) error {
	// 目标读取失败先返回，避免把不存在用户当成成功删除。
	user, err := s.store.GetUserByID(id)
	if err != nil {
		return mapUserLookupErr(err)
	}
	// 即使有其他管理员，当前目标仍须先失去管理员角色才能删除。
	if user.HasRole(RoleAdmin) {
		return ErrUserAdminCannotBeDeleted
	}
	// 只有存储删除成功才写入删除日志。
	err = s.store.DeleteUser(id)
	if err != nil {
		return err
	}
	s.log.Info("user deleted", "userID", id)
	return nil
}

// GetUsers 返回存储提供的全部内部用户对象，不在此处转换为公开用户结构。
func (s *UserService) GetUsers() ([]*User, error) {
	users, err := s.store.GetAllUsers()
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetUserByUsername 读取用户名对应账号，并按统一规则处理查找错误。
func (s *UserService) GetUserByUsername(username string) (*User, error) {
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		return nil, mapUserLookupErr(err)
	}
	return user, nil
}

// GetUserByIdentifier 先查用户名，任意失败后再把同一输入作为邮箱查询。
// 邮箱查询仍失败时才映射错误，不保留前一次用户名查询错误。
func (s *UserService) GetUserByIdentifier(identifier string) (*User, error) {
	user, err := s.store.GetUserByUsername(identifier)
	if err != nil {
		user, err = s.store.GetUserByEmail(identifier)
		if err != nil {
			return nil, mapUserLookupErr(err)
		}
	}
	return user, nil
}

// lookupRemoteUserByIdentifier 先查用户名，仅在明确不存在时继续查邮箱。
// 其他存储失败原样返回，使远端账号自动创建流程不会把查询故障当成缺少账号。
func (s *UserService) lookupRemoteUserByIdentifier(identifier string) (*User, error) {
	user, err := s.store.GetUserByUsername(identifier)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	return s.store.GetUserByEmail(identifier)
}

// GetOrCreateRemoteUser 查找反向代理提供的身份，不存在时用随机密码和指定角色创建账号。
// 已存在用户直接返回，不覆盖其邮箱或角色；新账号用户名原样使用 identifier。
// 邮箱为空时拼接 remote-user.invalid 占位地址，生成的随机明文不在结果中返回。
func (s *UserService) GetOrCreateRemoteUser(identifier, email, defaultRole string) (*User, error) {
	user, err := s.lookupRemoteUserByIdentifier(identifier)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	if email == "" {
		email = identifier + "@remote-user.invalid"
	}

	password, err := shared.GenerateRandomPassword(32)
	if err != nil {
		return nil, err
	}

	created, err := s.CreateUser(identifier, email, password, defaultRole)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			// 唯一性冲突后重新按身份查询，已经能解析账号则接受该结果。
			// 再查失败统一返回远端邮箱冲突错误，不把其他邮箱所属账号当作当前身份。
			if existing, lookupErr := s.lookupRemoteUserByIdentifier(identifier); lookupErr == nil {
				return existing, nil
			}
			return nil, ErrRemoteUserEmailConflict
		}
		return nil, err
	}

	s.log.Info("remote user auto-provisioned", "userID", created.ID, "username", identifier, "role", defaultRole)
	return created, nil
}

// GetUserByEmailOrUsernameAndPassword 先查用户名再回退邮箱，随后比较 bcrypt 摘要。
// 此方法不应用登录锁定计数，不生成令牌，成功返回完整内部用户对象。
func (s *UserService) GetUserByEmailOrUsernameAndPassword(identifier, password string) (*User, error) {
	user, err := s.store.GetUserByUsername(identifier)
	if err != nil {
		user, err = s.store.GetUserByEmail(identifier)
		if err != nil {
			return nil, mapUserLookupErr(err)
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, ErrUserInvalidCredentials
	}

	return user, nil
}

// ChangeOwnPassword 要求新旧明文不同并验证旧密码，然后保存新摘要。
// 不在方法内部检查最短长度或撤销会话，失败路径在写入前返回对应错误。
func (s *UserService) ChangeOwnPassword(id, oldPassword, newPassword string) error {
	if oldPassword == newPassword {
		return sharederrors.NewLocalizedError("auth_password_unchanged", "New password must differ from current password", "password unchanged", nil)
	}
	// 读取当前摘要用于校验，查找错误按服务统一规则映射。
	user, err := s.store.GetUserByID(id)
	if err != nil {
		return mapUserLookupErr(err)
	}

	// 旧密码不匹配时不生成新摘要，也不改变存储状态。
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
	if err != nil {
		return ErrUserInvalidCredentials
	}

	// 使用原样新密码生成 bcrypt 摘要。
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 存储成功后才记录密码变更日志，日志不包含密码本身。
	err = s.store.UpdatePassword(id, string(hashedPassword))
	if err != nil {
		return err
	}

	s.log.Info("password changed", "userID", id)
	return nil
}

// ResetAdminUserPassword 选择存储返回的管理员，按需创建或重命名，再设置恢复密码。
// 返回对象的 Password 是供维护调用方交付的明文，不能作为普通用户响应输出。
// 方法本身不撤销会话，维护入口在其成功后单独执行撤销。
func (s *UserService) ResetAdminUserPassword(username, email string, requestedPassword ...string) (*User, error) {
	// 可变参数只读取首项，非空显式密码必须达到字节长度阈值，空值生成十六字符密码。
	password := ""
	if len(requestedPassword) > 0 {
		password = requestedPassword[0]
	}
	if password != "" && len(password) < MinPasswordLength {
		return nil, ErrPasswordTooShort
	}
	if password == "" {
		generated, err := shared.GenerateRandomPassword(16)
		if err != nil {
			return nil, fmt.Errorf("failed to generate password: %w", err)
		}
		password = generated
	}

	// 只在存储直接返回 ErrUserNotFound 时创建管理员，其他错误立即返回。
	adminUser, err := s.store.GetAdminUser()
	if err != nil {
		if err == ErrUserNotFound {
			// 创建路径的空用户名和邮箱使用默认值，并经过 CreateUser 的角色配额检查。
			username = defaultIfEmpty(username, defaultAdminUsername)
			email = defaultIfEmpty(email, defaultAdminEmail)
			adminUser, err = s.CreateUser(username, email, password, RoleAdmin)
			if err != nil {
				return nil, fmt.Errorf("failed to create default admin: %w", err)
			}
		} else {
			return nil, err
		}
	}

	// 现有账号保留 ID 与邮箱，非空且不同的用户名交给 UpdateUser 检查唯一性后保存。
	// 重命名与密码保存是两个操作，后一步失败不撤销已经完成的重命名。
	if username != "" && username != adminUser.Username {
		updated, updateErr := s.UpdateUser(adminUser.ID, username, adminUser.Email, "", RoleAdmin)
		if updateErr != nil {
			return nil, updateErr
		}
		adminUser = updated
	}
	err = s.UpdatePassword(adminUser.ID, password)
	if err != nil {
		return nil, fmt.Errorf("failed to update admin password: %w", err)
	}

	// 返回维护结果时把摘要替换为明文，并把返回对象的强制改密状态设为 false。
	adminUser.Password = password
	adminUser.MustChangePassword = false

	s.log.Info("admin password reset", "userID", adminUser.ID)
	return adminUser, nil
}

// InviteUser 生成三十二字符随机密码创建账号，再单独设置待设密码标志。
// 随机明文不写日志也不返回，返回对象包含摘要；标志写入失败时账号已存在。
// 本方法只创建邀请状态，不发送邀请邮件或生成邮件令牌。
func (s *UserService) InviteUser(username, email, role string) (*User, error) {
	password, err := shared.GenerateRandomPassword(32)
	if err != nil {
		return nil, err
	}

	user, err := s.CreateUser(username, email, password, role)
	if err != nil {
		return nil, err
	}

	if err := s.store.SetMustSetPassword(user.ID, true); err != nil {
		return nil, err
	}
	user.MustSetPassword = true

	s.log.Info("user invited", "userID", user.ID, "role", user.Role)
	return user, nil
}

// CompleteInvite 先保存指定密码，再清除待设密码标志，供邮件邀请确认流程调用。
// 两次写入不共享此方法创建的事务，清除标志失败时密码已经保存。
func (s *UserService) CompleteInvite(id, password string) error {
	if err := s.UpdatePassword(id, password); err != nil {
		return err
	}
	if err := s.store.SetMustSetPassword(id, false); err != nil {
		return err
	}

	s.log.Info("invite accepted", "userID", id)
	return nil
}

// Close 释放用户存储持有的连接。
func (s *UserService) Close() error {
	return s.store.Close()
}

// suspendStore 交给存储关闭连接并标记暂停，供运行中的数据恢复流程阻止继续访问。
func (s *UserService) suspendStore() error {
	return s.store.suspend()
}
