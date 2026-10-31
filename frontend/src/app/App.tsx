// 应用根组件装配注册表、运行配置、会话初始化、设置同步和路由容器。
import '@/app/registrations'
import ForcePasswordChange from '@/features/auth/ForcePasswordChange'
import { startSettingsSync } from '@/app/settingsSync'
import { ErrorBoundary } from '@/shared/components/ErrorBoundary'
import { createLeafDocRouter } from '@/app/router/router'
import { useBootstrapAuth } from '@/app/bootstrapAuth'
import { BASE_PATH } from '@/shared/lib/config'
import { useFavoritesStore } from '@/features/favorites/state/favorites'
import { useSessionStore } from '@/shared/state/session'
import useApplyDesignMode from '@/app/useApplyDesignMode'
import { Loader2 } from 'lucide-react'
import { Suspense, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { RouterProvider } from 'react-router/dom'
import { toast, Toaster } from 'sonner'
import { useTranslation } from 'react-i18next'
import { useBrandingStore } from '@/features/branding/state/branding'
import { useConfigStore } from '@/shared/state/config'
import { useRestoreStore } from '@/features/restore/state/restore'

// App 在配置尝试结束且会话不处于刷新状态时显示主体，强制改密状态优先于路由。
function App() {
  const { t } = useTranslation('common')
  const configHasLoaded = useConfigStore((s) => s.hasLoaded)
  const configError = useConfigStore((s) => s.error)
  const loadConfig = useConfigStore((s) => s.loadConfig)
  const enableRevision = useConfigStore((s) => s.enableRevision)
  const loginUrl = useConfigStore((s) => s.loginUrl)
  const smtpEnabled = useConfigStore((s) => s.smtpEnabled)
  const loadBranding = useBrandingStore((s) => s.loadBranding)
  const lastConfigErrorRef = useRef<string | null>(null)

  // 配置尝试结束后启用身份初始化，真实配置成功标志由该 Hook 内部另行读取。
  useBootstrapAuth(configHasLoaded)

  const userId = useSessionStore((s) => s.user?.id ?? null)
  const isRefreshing = useSessionStore((s) => s.isRefreshing)
  const loadFavorites = useFavoritesStore((s) => s.loadFavorites)
  const clearFavorites = useFavoritesStore((s) => s.clearFavorites)

  const mustChangePassword = useSessionStore((s) => s.user?.mustChangePassword)
  const isRestoring = useRestoreStore((s) => s.isLoading || s.requiresLogin)
  // 恢复处理中或要求重新登录时停止设置同步，恢复条件解除后重新建立同步生命周期。
  useEffect(() => {
    if (!isRestoring) return startSettingsSync()
  }, [isRestoring])
  useApplyDesignMode()
  useEffect(() => {
    loadConfig()
  }, [loadConfig])

  // 用户 ID 或强制改密状态变化时读取或清空收藏，匿名和待改密状态不保留收藏展示。
  useEffect(() => {
    if (userId && !mustChangePassword) {
      loadFavorites()
    } else {
      clearFavorites()
    }
  }, [userId, mustChangePassword, loadFavorites, clearFavorites])

  useLayoutEffect(() => {
    // 布局副作用中发起品牌配置加载，不阻塞等待请求完成。
    loadBranding()
  }, [loadBranding])

  // 相同错误连续出现时只显示一次通知，错误消失后允许同一消息再次显示。
  useEffect(() => {
    if (!configError) {
      lastConfigErrorRef.current = null
      return
    }

    if (lastConfigErrorRef.current === configError) return

    lastConfigErrorRef.current = configError
    toast.error(configError)
  }, [configError])

  const [router, setRouter] = useState<ReturnType<
    typeof createLeafDocRouter
  > | null>(null)
  // 配置相关路由选项变化时创建新实例，清理时 dispose 释放其历史监听等资源。
  // configHasLoaded 包含最终失败状态，并不单独要求配置请求成功。
  useEffect(() => {
    if (!configHasLoaded) return
    const instance = createLeafDocRouter(
      enableRevision,
      loginUrl,
      smtpEnabled,
      BASE_PATH || undefined,
    )
    setRouter(instance)
    return () => instance.dispose()
  }, [configHasLoaded, enableRevision, loginUrl, smtpEnabled])

  return (
    <>
      <Toaster
        offset={16}
        mobileOffset={16}
        richColors
        closeButton
        position="bottom-right"
        containerAriaLabel={t('notifications')}
      />
      {configHasLoaded && router && !isRefreshing ? (
        <ErrorBoundary>
          <Suspense
            fallback={
              <div className="flex h-screen items-center justify-center">
                <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
              </div>
            }
          >
            {mustChangePassword ? (
              <ForcePasswordChange />
            ) : (
              <RouterProvider router={router} />
            )}
          </Suspense>
        </ErrorBoundary>
      ) : null}
    </>
  )
}

export default App
