// 此表单提交账号标识并展示统一确认，不向用户区分该标识是否对应真实账号。
import { mapApiError } from '@/shared/lib/api/errors'
import { toast } from 'sonner'
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { requestPasswordReset } from '@/shared/lib/api/auth'
import { useBrandingStore } from '@/features/branding/state/branding'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'

// 站点名称取自品牌状态，提交和确认页面共用同一个浏览器标题。
export default function ForgotPasswordForm() {
  const { t } = useTranslation('auth')
  const { siteName } = useBrandingStore()
  const [identifier, setIdentifier] = useState('')
  const [loading, setLoading] = useState(false)
  // 请求成功后固定进入确认分支，不从响应推断邮箱是否存在或邮件是否送达。
  const [submitted, setSubmitted] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    // 禁止浏览器表单导航，等待 API 后切换状态，失败保留输入并映射错误提示。
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
    // 确认页不再提供重复提交表单，只提供返回登录入口。
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
