// 通过设置事件长轮询和页面重新可见事件刷新实例配置、品牌及管理员用户列表。
import { toast } from 'sonner'
import i18next from '@/shared/lib/i18n'
import { useSessionStore } from '@/shared/state/session'
import { useUserStore } from '@/features/users/state/users'
import { API_BASE_URL } from '@/shared/lib/config'
import { useConfigStore } from '@/shared/state/config'
import { useBrandingStore } from '@/features/branding/state/branding'
// startSettingsSync 立即发起轮询并返回清理函数，每次调用维护独立游标和请求状态。
export function startSettingsSync() {
  const retryDelayMs = 3000
  let cursor = ''
  let retryTimer: ReturnType<typeof setTimeout> | undefined
  let running = false,
    again = false,
    stopped = false
  // 刷新串行执行，并把执行中再次触发的刷新合并为下一轮，不建立无限请求队列。
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
        // 用户列表只在当前身份为管理员且无需改密时读取。
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
  // 每轮完整解析 JSON 后才继续下一轮，keepalive 请求允许浏览器在页面离开时继续处理。
  // 清理不使用 AbortController，进行中的请求读完后按 stopped 标志停止后续动作。
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
      // 只接受带非空字符串 cursor 的对象，其他响应作为失败进入延迟重试。
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
      // 游标变化才刷新数据，成功响应未变化时直接继续轮询，不额外等待。
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
      // 请求或解析失败显示通知并等待固定三秒，清理函数会取消尚未触发的定时器。
      retryTimer = setTimeout(() => {
        void poll()
      }, retryDelayMs)
    }
  }
  void poll()
  // 标签页重新可见时主动刷新，不依赖下一次事件游标变化。
  const visible = () => {
    if (document.visibilityState === 'visible') void refresh()
  }
  document.addEventListener('visibilitychange', visible)
  // 标记停止、移除可见性监听并清理重试计时，已开始的刷新不会在此强制取消。
  return () => {
    stopped = true
    clearTimeout(retryTimer)
    document.removeEventListener('visibilitychange', visible)
  }
}
