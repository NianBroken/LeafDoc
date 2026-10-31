// 认证、用户与 API 密钥模块共用角色取值，字符串与后端角色值对应。
export type UserRole = 'admin' | 'editor' | 'viewer'
// SessionUser 是浏览器会话使用的不带密码用户信息，包含邮箱及可选强制改密标志。
// 可选标志允许缺失该字段的响应，类型声明本身不进行权限检查或数据校验。
export type SessionUser = {
  id: string
  username: string
  email: string
  role: UserRole
  mustChangePassword?: boolean
}
