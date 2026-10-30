import { mapApiError } from '@/shared/lib/api/errors'
import { toast } from 'sonner'
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { requestPasswordReset } from '@/shared/lib/api/auth'
import { useBrandingStore } from '@/features/branding/state/branding'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'

export default function ForgotPasswordForm() {
  const { t } = useTranslation('auth')
  const { siteName } = useBrandingStore()
  const [identifier, setIdentifier] = useState('')
  const [loading, setLoading] = useState(false)
  // Once true, the form always shows the same generic confirmation — the
  // backend deliberately returns identical success responses whether or not
  // the identifier resolved to a real account (see requestPasswordReset's
  // doc comment), so the UI must not try to distinguish the two either.
  const [submitted, setSubmitted] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      await requestPasswordReset(identifier)
      setSubmitted(true)
    } catch (err) {
      toast.error(mapApiError(err, t('apiErrors.requestFailed')).message)
    } finally {
      setLoading(false)
    }
  }

  if (submitted) {
    return (
      <>
        <title>{t('forgotPassword.pageTitle', { siteName })}</title>
        <div className="login">
          <div className="login__form">
            <h1 className="login__title">
              {t('forgotPassword.confirmationTitle')}
            </h1>
            <p className="login__description">
              {t('forgotPassword.confirmationDescription')}
            </p>
            <Link to="/login">
              <Button variant="ghost" className="login__back">
                {t('forgotPassword.backToLogin')}
              </Button>
            </Link>
          </div>
        </div>
      </>
    )
  }

  return (
    <>
      <title>{t('forgotPassword.pageTitle', { siteName })}</title>
      <div className="login">
        <form onSubmit={handleSubmit} className="login__form">
          <h1 className="login__title">{t('forgotPassword.title')}</h1>
          <p className="login__description">
            {t('forgotPassword.description')}
          </p>

          <div className="login__field">
            <Input
              type="text"
              placeholder={t('forgotPassword.identifierPlaceholder')}
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              required
              name="identifier"
              autoComplete="username"
              autoFocus
              data-testid="forgot-password-identifier"
              spellCheck={false}
            />
          </div>

          <Button
            type="submit"
            className="login__submit"
            disabled={loading}
            data-testid="forgot-password-submit"
          >
            {loading
              ? t('forgotPassword.submitting')
              : t('forgotPassword.submit')}
          </Button>
          <Link to="/login">
            <Button variant="ghost" className="login__back">
              {t('forgotPassword.backToLogin')}
            </Button>
          </Link>
        </form>
      </div>
    </>
  )
}
