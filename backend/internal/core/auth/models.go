// 本文件定义认证用户的内部表示、接口表示及角色取值。
package auth

// PublicUser 是不带密码字段的用户响应结构，仍包含邮箱和权限状态。
// 结构本身不执行访问控制，是否向某个请求返回这些字段由接口层决定。
type PublicUser struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"mustChangePassword"`
	MustSetPassword    bool   `json:"mustSetPassword"`
}

// User 是认证服务的内部用户结构，Password 在不同调用路径中承载摘要或临时明文。
// 对外用户结果通过 ToPublicUser 生成，不直接复制敏感密码字段。
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`

	// MustSetPassword 表示邀请账号尚待设置密码。InviteUser 为账号创建随机密码，
	// CompleteInvite 在保存用户密码后清除此标志，前端据此显示邀请状态。
	MustSetPassword bool

	// MustChangePassword 传达账号需要更换密码的状态，具体请求限制由上层执行。
	MustChangePassword bool
}

// HasRole 精确比较角色字符串，不执行角色高低关系判断或大小写转换。
func (u *User) HasRole(role string) bool {
	return u.Role == role
}

// ToPublicUser 复制身份和状态字段到新对象，不复制密码，也不修改原对象。
func (u *User) ToPublicUser() *PublicUser {
	return &PublicUser{
		ID:                 u.ID,
		Username:           u.Username,
		Email:              u.Email,
		Role:               u.Role,
		MustChangePassword: u.MustChangePassword,
		MustSetPassword:    u.MustSetPassword,
	}
}

// 角色常量同时用于存储和权限判断，取值区分大小写。
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// validRoles 集中列出受支持角色，不为未知角色提供默认权限。
var validRoles = map[string]bool{
	RoleAdmin:  true,
	RoleEditor: true,
	RoleViewer: true,
}

// IsValidRole 仅接受角色表中的精确匹配值，空值和未知值返回 false。
func IsValidRole(role string) bool {
	return validRoles[role]
}
