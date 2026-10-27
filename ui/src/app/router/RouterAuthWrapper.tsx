// Access checks follow live session state without replacing the browser router.
import { ErrorBoundary } from '@/shared/components/ErrorBoundary'
import RequireAuth from '@/features/auth/RequireAuth'
import AppLayout from '@/app/layout/AppLayout'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { useSessionStore } from '@/shared/state/session'
import { Navigate, useLocation } from 'react-router'
import type { ReactNode } from 'react'

export default function AuthWrapper({
  children,
  allowPublic = false,
}: {
  children: ReactNode
  allowPublic?: boolean
}) {
  const { pathname } = useLocation()
  const user = useSessionStore((state) => state.user)
  const isReadOnly = useIsReadOnly()
  const publicViewer = isReadOnly && !user
  if (publicViewer && !allowPublic) return <Navigate to="/" replace />
  const content = <AppLayout>{children}</AppLayout>
  return (
    <ErrorBoundary resetKey={pathname}>
      {publicViewer ? content : <RequireAuth>{content}</RequireAuth>}
    </ErrorBoundary>
  )
}
