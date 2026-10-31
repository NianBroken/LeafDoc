// 本文件封装用户管理和本人改密请求，不自行判断管理员权限或改变前端会话。
import type { SessionUser } from '@/shared/contracts/auth'
import { fetchWithAuth } from '@/shared/lib/api/client'

export type User = SessionUser & {
  // 表示用户尚需设置密码的服务端状态，不在客户端依据邀请历史自行推断。
  mustSetPassword: boolean
}

// UserInput 从用户结构选取名称、邮箱和角色，密码在各请求参数中单独声明。
// 这种静态类型约束不会在运行时删除输入对象中额外存在的字段。
type UserInput = Pick<User, 'username' | 'email' | 'role'>

// 用户创建与邮件发送结果分开报告，调用方需分别展示用户和邮件状态。
export type InviteUserResponse = {
  user: User
  emailSent: boolean
}

// 获取用户列表，不在此过滤角色或排序名称。
export async function getUsers(): Promise<User[]> {
  return await fetchWithAuth<User[]>('/api/users')
}

// 将包含显式密码的用户输入作为 JSON 提交，服务端负责校验和散列存储。
export async function createUser(user: UserInput & { password: string }) {
  return await fetchWithAuth('/api/users', {
    method: 'POST',
    body: JSON.stringify(user),
  })
}

// inviteUser 向邀请接口提交不含显式密码的账号字段，返回用户和 emailSent。
// 本层不把 emailSent 为 false 转成请求失败，界面根据该字段决定如何提示。
export async function inviteUser(user: UserInput): Promise<InviteUserResponse> {
  return await fetchWithAuth<InviteUserResponse>('/api/users/invite', {
    method: 'POST',
    body: JSON.stringify(user),
  })
}

// 对指定用户重发邀请，不在请求前确认邀请状态或自动创建账号。
export async function resendInvite(id: string) {
  return await fetchWithAuth(`/api/users/${id}/invite/resend`, {
    method: 'POST',
  })
}

// 用户 ID 直接拼接路径，同时整个输入对象编码到正文，包括 id 和可选 password。
export async function updateUser(
  user: UserInput & { id: string; password?: string },
) {
  return await fetchWithAuth(`/api/users/${user.id}`, {
    method: 'PUT',
    body: JSON.stringify(user),
  })
}

// 本人改密同时提交旧密码和新密码，不在本层验证复杂度或清空登录状态。
export async function changeOwnPassword(
  oldPassword: string,
  newPassword: string,
) {
  return await fetchWithAuth(`/api/users/me/password`, {
    method: 'PUT',
    body: JSON.stringify({
      oldPassword,
      newPassword,
    }),
  })
}

// 删除请求原样使用用户标识，关联资源和会话清理由服务端决定。
export async function deleteUser(id: string) {
  return await fetchWithAuth(`/api/users/${id}`, {
    method: 'DELETE',
  })
}
