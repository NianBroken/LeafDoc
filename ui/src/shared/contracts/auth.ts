// User roles and session identity are shared by authentication, users and API keys.
export type UserRole = 'admin' | 'editor' | 'viewer'
export type SessionUser = {
  id: string
  username: string
  email: string
  role: UserRole
  mustChangePassword?: boolean
}
