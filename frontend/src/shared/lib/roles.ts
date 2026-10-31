// 组件角色门禁和设置项注册共用的角色列表判断，不在此验证会话真实性。
// role 为空时拒绝，允许列表为空时接受任何非空角色，否则要求精确包含该角色。
// 未知非空角色也能通过空列表，实际角色合法性由会话来源保证。

export function hasRole(
  role: string | undefined,
  allowedRoles: string[],
): boolean {
  if (!role) return false
  if (allowedRoles.length === 0) return true
  return allowedRoles.includes(role)
}
