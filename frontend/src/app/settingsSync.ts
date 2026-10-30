import { toast } from 'sonner'
import i18next from '@/shared/lib/i18n'
import { useSessionStore } from '@/shared/state/session'
import { useUserStore } from '@/features/users/state/users'
import { API_BASE_URL } from '@/shared/lib/config'
import { useConfigStore } from '@/shared/state/config'
import { useBrandingStore } from '@/features/branding/state/branding'
export function startSettingsSync() {
  const retryDelayMs = 3000
  let cursor = ''
  let retryTimer: ReturnType<typeof setTimeout> | undefined
  let running = false,
    again = false,
    stopped = false
  const refresh = async () => {
    if (stopped) return
    if (running) {
      again = true
      return
    }
    running = true
    try {
      do {
        again = false
        await useConfigStore.getState().loadConfig()
        if (stopped) return
        await useBrandingStore.getState().loadBranding()
        if (stopped) return
        const user = useSessionStore.getState().user
        if (user?.role === 'admin' && !user.mustChangePassword)
          await useUserStore.getState().loadUsers()
      } while (again && !stopped)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : i18next.t('configLoad.errorFallback', { ns: 'common' }),
      )
    } finally {
      running = false
    }
  }
  // Complete bounded requests even when the view is unloaded. No streaming body
  // remains half-read at the proxy, and at most one request is active per view.
  const poll = async () => {
    try {
      const query = new URLSearchParams({ poll: 'true', cursor })
      const response = await fetch(
        `${API_BASE_URL}/api/settings/events?${query}`,
        {
          credentials: 'include',
          keepalive: true,
        },
      )
      if (!response.ok)
        throw new Error(`Settings sync: HTTP ${response.status}`)
      const next: unknown = await response.json()
      if (
        typeof next !== 'object' ||
        next === null ||
        !('cursor' in next) ||
        typeof next.cursor !== 'string' ||
        !next.cursor
      )
        throw new Error('Settings sync: invalid response')
      if (stopped) return
      if (next.cursor !== cursor) {
        cursor = next.cursor
        await refresh()
      }
      if (!stopped) void poll()
    } catch (error) {
      if (stopped) return
      toast.error(
        error instanceof Error
          ? error.message
          : i18next.t('configLoad.errorFallback', { ns: 'common' }),
      )
      retryTimer = setTimeout(() => {
        void poll()
      }, retryDelayMs)
    }
  }
  void poll()
  const visible = () => {
    if (document.visibilityState === 'visible') void refresh()
  }
  document.addEventListener('visibilitychange', visible)
  return () => {
    stopped = true
    clearTimeout(retryTimer)
    document.removeEventListener('visibilitychange', visible)
  }
}
