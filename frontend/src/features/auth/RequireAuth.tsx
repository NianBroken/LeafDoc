import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router'
import ExternalRedirect from '@/features/auth/ExternalRedirect'

type Props = {
  children: ReactNode
}

export default function RequireAuth({ children }: Props) {
  const user = useSessionStore((state) => state.user)
  const isRefreshing = useSessionStore((state) => state.isRefreshing)
  const loginUrl = useConfigStore((state) => state.loginUrl)
  const location = useLocation()

  if (!user && !isRefreshing) {
    const redirectTo = `${location.pathname}${location.search}${location.hash}`
    if (loginUrl) {
      return <ExternalRedirect to={loginUrl} returnTo={redirectTo} />
    }
    return <Navigate to="/login" replace state={{ redirectTo }} />
  }

  if (!user && isRefreshing) {
    return null // Return a loading state while refreshing
  }

  return <>{children}</>
}
