// 本文件按认证模式和缓存令牌期限选择身份查询或刷新，并同步会话加载状态。
import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { useEffect } from 'react'
import { fetchMe } from '@/shared/lib/api/auth'
import { ApiError, ensureRefresh } from '@/shared/lib/api/client'

// useBootstrapAuth 在依赖变化后启动一次异步初始化，清理标志阻止本 Hook 的迟到状态写入。
// 它不取消底层请求，ensureRefresh 自身产生的共享状态更新也不受此局部标志控制。
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
        // 到期时间以秒保存，只有严格晚于当前时间才视为仍有效。
        const accessIsCurrent =
          (session.accessTokenExpiresAt ?? 0) > Date.now() / 1000
        if (
          httpRemoteUserEnabled ||
          !configLoadSucceeded ||
          !session.user ||
          accessIsCurrent
        ) {
          // 代理认证、配置未确认、无缓存用户或访问令牌仍有效时直接读取当前身份。
          // 仅已确认普通模式且缓存用户的令牌过期时，进入刷新分支。
          const user = await fetchMe()
          if (!cancelled) setUser(user)
        } else {
          await ensureRefresh()
        }
      } catch (err) {
        // 只对 ApiError 的 400 至 499 状态清除用户，其他错误保留缓存身份。
        // 此分类包含全部 4xx，不在这里区分未授权、限流或其他客户端错误。
        const isAuthFailure =
          err instanceof ApiError && err.status >= 400 && err.status < 500
        if (!cancelled && isAuthFailure) setUser(null)
        console.debug('[bootstrapAuth] Auth bootstrap failed:', err)
      } finally {
        if (!cancelled) setRefreshing(false)
      }
    })()

    // 依赖变化或组件卸载后，局部异步任务不再写用户和刷新结束标志。
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
