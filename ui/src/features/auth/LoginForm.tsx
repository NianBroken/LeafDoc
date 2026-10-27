import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { login } from '@/shared/lib/api/auth'
import { mapApiError } from '@/shared/lib/api/errors'
import { withBasePath } from '@/shared/lib/routePath'
import { useBrandingStore } from '@/features/branding/state/branding'
import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Navigate, useLocation, useNavigate } from 'react-router'
import { toast } from 'sonner'

function getRedirectTo(state: unknown): string | null {
  if (!state || typeof state !== 'object') {
    return null
  }

  const redirectTo = (state as { redirectTo?: unknown }).redirectTo
  if (typeof redirectTo !== 'string' || !redirectTo.startsWith('/')) {
    return null
  }

  if (
    redirectTo === '/login' ||
    redirectTo.startsWith('/login?') ||
    redirectTo.startsWith('/login#')
  ) {
    return null
  }

  return redirectTo
}

export default function LoginForm() {
  const { t } = useTranslation('auth')
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  const location = useLocation()
  const navigate = useNavigate()
  const user = useSessionStore((s) => s.user)
  const smtpEnabled = useConfigStore((s) => s.smtpEnabled)
  const { siteName, logoFile, logoVersion } = useBrandingStore()
  const redirectTo = getRedirectTo(location.state)

  // If already logged in, redirect to home
  if (user) {
    return <Navigate to={redirectTo || '/'} replace />
  }

  const handleCredentialsSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      await login(identifier, password)
      // user already set in the store by the login function
      // Restore the originally requested page after successful login.
      navigate(redirectTo || '/', { replace: true })
    } catch (err) {
      const mapped = mapApiError(err, t('login.errorFallback'))
      toast.error(mapped.message)
    } finally {
      setLoading(false)
    }
  }

  const logoHeader = (
    <h1 className="login__title">
      {logoFile ? (
        <img
          src={`${withBasePath(`/branding/${logoFile}`)}?v=${logoVersion}`}
          alt={siteName}
          className="login__logo-image"
        />
      ) : (
        <span>🌿</span>
      )}{' '}
      {siteName}
    </h1>
  )

  return (
    <>
      <title>{t('login.pageTitle', { siteName })}</title>
      <div className="login">
        <form onSubmit={handleCredentialsSubmit} className="login__form">
          {logoHeader}

          <div className="login__field">
            <Input
              type="text"
              placeholder={t('login.identifierPlaceholder')}
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              required
              name="identifier"
              autoComplete="username"
              data-testid="login-identifier"
              spellCheck={false}
            />
          </div>

          <div className="login__field">
            <Input
              type="password"
              placeholder={t('login.passwordPlaceholder')}
              name="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              data-testid="login-password"
              autoComplete="current-password"
              spellCheck={false}
            />
          </div>

          {smtpEnabled && (
            <Link
              to="/forgot-password"
              className="login__forgot-password"
              data-testid="login-forgot-password-link"
            >
              {t('login.forgotPasswordLink')}
            </Link>
          )}

          <Button
            type="submit"
            className="login__submit"
            disabled={loading}
            data-testid="login-submit"
          >
            {loading ? t('login.submitting') : t('login.submit')}
          </Button>
        </form>
      </div>
    </>
  )
}
