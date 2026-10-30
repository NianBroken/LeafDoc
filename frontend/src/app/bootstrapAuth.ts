import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { useEffect } from 'react'
import { fetchMe } from '@/shared/lib/api/auth'
import { ApiError, ensureRefresh } from '@/shared/lib/api/client'

export function useBootstrapAuth(enabled = true) {
  const setUser = useSessionStore((s) => s.setUser)
  const setRefreshing = useSessionStore((s) => s.setRefreshing)
  const httpRemoteUserEnabled = useConfigStore((s) => s.httpRemoteUserEnabled)
  const configLoadSucceeded = useConfigStore((s) => s.configLoadSucceeded)

  useEffect(() => {
    if (!enabled) {
      setRefreshing(false)
      return
    }
    let cancelled = false

    ;(async () => {
      setRefreshing(true)
      try {
        const session = useSessionStore.getState()
        const accessIsCurrent =
          (session.accessTokenExpiresAt ?? 0) > Date.now() / 1000
        if (
          httpRemoteUserEnabled ||
          !configLoadSucceeded ||
          !session.user ||
          accessIsCurrent
        ) {
          // Identity reads do not rotate cookies. Fresh sessions and anonymous
          // visits use this path so navigation cannot interrupt a token rotation.
          // Proxy-managed sessions also resolve identity without JWT refresh.
          const user = await fetchMe()
          if (!cancelled) setUser(user)
        } else {
          await ensureRefresh()
        }
      } catch (err) {
        // Only clear the session for explicit auth failures (4xx).
        // Server errors (5xx) and network failures must not log the user out —
        // the backend returns 200 for both authenticated and unauthenticated
        // states, so a throw always indicates a genuine server problem.
        const isAuthFailure =
          err instanceof ApiError && err.status >= 400 && err.status < 500
        if (!cancelled && isAuthFailure) setUser(null)
        console.debug('[bootstrapAuth] Auth bootstrap failed:', err)
      } finally {
        if (!cancelled) setRefreshing(false)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [
    setUser,
    setRefreshing,
    enabled,
    httpRemoteUserEnabled,
    configLoadSucceeded,
  ])
}
