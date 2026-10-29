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

function App() {
  const { t } = useTranslation('common')
  const configHasLoaded = useConfigStore((s) => s.hasLoaded)
  const configError = useConfigStore((s) => s.error)
  const loadConfig = useConfigStore((s) => s.loadConfig)
  const authDisabled = useConfigStore((s) => s.authDisabled)
  const enableRevision = useConfigStore((s) => s.enableRevision)
  const loginUrl = useConfigStore((s) => s.loginUrl)
  const smtpEnabled = useConfigStore((s) => s.smtpEnabled)
  const loadBranding = useBrandingStore((s) => s.loadBranding)
  const lastConfigErrorRef = useRef<string | null>(null)

  // bootstrap authentication on app start -> session store
  useBootstrapAuth(configHasLoaded && !authDisabled)

  const userId = useSessionStore((s) => s.user?.id ?? null)
  const isRefreshing = useSessionStore((s) => s.isRefreshing)
  const loadFavorites = useFavoritesStore((s) => s.loadFavorites)
  const clearFavorites = useFavoritesStore((s) => s.clearFavorites)

  const mustChangePassword = useSessionStore((s) => s.user?.mustChangePassword)
  const isRestoring = useRestoreStore((s) => s.isLoading || s.requiresLogin)
  useEffect(() => {
    if (!isRestoring) return startSettingsSync()
  }, [isRestoring])
  useApplyDesignMode()
  useEffect(() => {
    loadConfig()
  }, [loadConfig])

  // Favorites are per-user server truth — (re)load whenever the logged-in
  // user changes, and clear them on logout so a second user on the same
  // browser never sees the first user's favorites.
  useEffect(() => {
    if (userId && !mustChangePassword) {
      loadFavorites()
    } else {
      clearFavorites()
    }
  }, [userId, mustChangePassword, loadFavorites, clearFavorites])

  useLayoutEffect(() => {
    // Load branding configuration
    loadBranding()
  }, [loadBranding])

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
  // Router ownership follows configuration readiness and releases history listeners on cleanup.
  useEffect(() => {
    if (!configHasLoaded) return
    const instance = createLeafDocRouter(
      authDisabled,
      enableRevision,
      loginUrl,
      smtpEnabled,
      BASE_PATH || undefined,
    )
    setRouter(instance)
    return () => instance.dispose()
  }, [configHasLoaded, authDisabled, enableRevision, loginUrl, smtpEnabled])

  return (
    <>
      <Toaster
        richColors
        position="bottom-right"
        containerAriaLabel={t('notifications')}
      />
      {configHasLoaded && router && !(isRefreshing && !authDisabled) ? (
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
