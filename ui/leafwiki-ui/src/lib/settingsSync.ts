import { toast } from 'sonner'
import i18next from '@/lib/i18n'
import { useSessionStore } from '@/stores/session'
import { useUserStore } from '@/stores/users'
// EventSource reconnects automatically; every reconnect fetches authoritative shared state.
import { API_BASE_URL } from '@/lib/config'
import { useConfigStore } from '@/stores/config'
import { useBrandingStore } from '@/stores/branding'
export function startSettingsSync() {
  const source = new EventSource(`${API_BASE_URL}/api/settings/events`, {
    withCredentials: true,
  })
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
        await useBrandingStore.getState().loadBranding()
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
  source.onmessage = () => {
    void refresh()
  }
  const visible = () => {
    if (document.visibilityState === 'visible') void refresh()
  }
  document.addEventListener('visibilitychange', visible)
  return () => {
    stopped = true
    source.close()
    document.removeEventListener('visibilitychange', visible)
  }
}
