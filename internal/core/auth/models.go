package auth

// PublicUser represents a user object that is safe to expose to the public.
type PublicUser struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"mustChangePassword"`
	MustSetPassword    bool   `json:"mustSetPassword"`
}

// User represents a user object with sensitive information.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`

	// MustSetPassword is true for a user created via InviteUser who hasn't
	// yet accepted their invite (see UserService.InviteUser/CompleteInvite):
	// they have a real row and a random, never-returned password, but cannot
	// meaningfully log in until they set their own.
	MustSetPassword bool

	// MustChangePassword restricts a bootstrap account until its password is changed.
	MustChangePassword bool
}

func (u *User) HasRole(role string) bool {
	return u.Role == role
}

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

// Roles
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

var validRoles = map[string]bool{
	RoleAdmin:  true,
	RoleEditor: true,
	RoleViewer: true,
}

func IsValidRole(role string) bool {
	return validRoles[role]
}
