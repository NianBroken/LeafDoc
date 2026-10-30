// Renders children only for a role accepted by the shared permission rules.

import { hasRole } from '@/shared/lib/roles'
import { useSessionStore } from '@/shared/state/session'
import { ReactNode } from 'react'

type Props = {
  roles: string[]
  children: ReactNode
}

export function RoleGuard({ roles, children }: Props) {
  const user = useSessionStore((state) => state.user)

  if (!hasRole(user?.role, roles)) return null

  return <>{children}</>
}
