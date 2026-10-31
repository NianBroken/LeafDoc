// 此显示门禁复用共享角色规则，不满足条件时不渲染内容，不替代服务器权限检查。

import { hasRole } from '@/shared/lib/roles'
import { useSessionStore } from '@/shared/state/session'
import { ReactNode } from 'react'

// roles 是允许角色清单，空清单接受任意非空角色，未登录用户始终被拒绝。
type Props = {
  roles: string[]
  children: ReactNode
}

// 从当前会话读取角色，变化后重新判断，不在本组件发起认证请求。
export function RoleGuard({ roles, children }: Props) {
  const user = useSessionStore((state) => state.user)

  if (!hasRole(user?.role, roles)) return null

  return <>{children}</>
}
