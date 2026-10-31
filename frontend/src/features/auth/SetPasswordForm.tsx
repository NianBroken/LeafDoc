// 本文件共用邀请接受与密码重置输入流程，依据模式调用不同接口并处理成功页面。
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { acceptInvite, confirmPasswordReset } from '@/shared/lib/api/auth'
import { handleFieldErrors } from '@/shared/lib/handleFieldErrors'
import { useBrandingStore } from '@/features/branding/state/branding'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router'

// reset 完成后展示确认，invite 完成后由 API 应用会话并导航首页。
type SetPasswordFormProps = {
  mode: 'reset' | 'invite'
}

// 此处以 JavaScript 字符串长度检查八个 UTF-16 代码单元，提前显示本地化提示。
// 后端独立按 Go 字符串字节数校验，两者对非 ASCII 输入的计数单位不同。
const MIN_PASSWORD_LENGTH = 8

// SetPasswordForm 从查询参数读取 token，缺少或空令牌时直接替换导航到登录页。
// 重置确认 API 不应用会话；邀请接受 API 应用返回的认证信息，此处据此选择成功行为。
export function SetPasswordForm({ mode }: SetPasswordFormProps) {
  const { t } = useTranslation('auth')
  const { siteName } = useBrandingStore()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''

  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(false)
  const [resetSucceeded, setResetSucceeded] = useState(false)

  const ns = mode === 'reset' ? 'resetPassword' : 'acceptInvite'

  if (!token) {
    return <Navigate to="/login" replace />
  }

  const handleSubmit = async (e: React.FormEvent) => {
    // 每次提交清空已有字段错误，长度与两次输入一致性通过后才设置加载状态并请求。
    e.preventDefault()
    setFieldErrors({})

    if (newPassword.length < MIN_PASSWORD_LENGTH) {
      setFieldErrors({ newPassword: t(`${ns}.passwordTooShort`) })
      return
    }

    // 请求只发送 newPassword，确认输入不发给后端，因此一致性由表单校验。
    if (newPassword !== confirmPassword) {
      setFieldErrors({ confirmPassword: t(`${ns}.passwordsDoNotMatch`) })
      return
    }

    setLoading(true)
    try {
      if (mode === 'reset') {
        await confirmPasswordReset(token, newPassword)
        setResetSucceeded(true)
      } else {
        await acceptInvite(token, newPassword)
        // acceptInvite 已写入会话，替换导航避免后退再次展示携带令牌的表单。
        navigate('/', { replace: true })
      }
    } catch (err) {
      // 字段校验错误写到对应输入下方，其他错误交给公共处理器显示提示。
      handleFieldErrors(err, setFieldErrors, t(`${ns}.errorFallback`))
    } finally {
      setLoading(false)
    }
  }

  if (resetSucceeded) {
    // 重置成功后不继续显示令牌输入流程，提供显式返回登录的入口。
    return (
      <>
        <title>{t('resetPassword.pageTitle', { siteName })}</title>
        <div className="login">
          <div className="login__form">
            <h1 className="login__title">{t('resetPassword.successTitle')}</h1>
            <p className="login__description">
              {t('resetPassword.successDescription')}
            </p>
            <Link to="/login">
              <Button variant="ghost" className="login__back">
                {t('resetPassword.goToLogin')}
              </Button>
            </Link>
          </div>
        </div>
      </>
    )
  }

  return (
    <>
      <title>{t(`${ns}.pageTitle`, { siteName })}</title>
      <div className="login">
        <form onSubmit={handleSubmit} className="login__form">
          <h1 className="login__title">{t(`${ns}.title`)}</h1>
          {ns === 'acceptInvite' && (
            <p className="login__description">
              {t('acceptInvite.description')}
            </p>
          )}

          <div className="login__field">
            <Input
              type="password"
              placeholder={t(`${ns}.newPasswordPlaceholder`)}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              name="new-password"
              autoComplete="new-password"
              autoFocus
              data-testid="set-password-new"
              spellCheck={false}
            />
            {fieldErrors.newPassword && (
              <p className="text-error mt-1 text-sm">
                {fieldErrors.newPassword}
              </p>
            )}
          </div>
          <div className="login__field">
            <Input
              type="password"
              placeholder={t(`${ns}.confirmPasswordPlaceholder`)}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              name="confirm-password"
              autoComplete="new-password"
              data-testid="set-password-confirm"
              spellCheck={false}
            />
            {fieldErrors.confirmPassword && (
              <p className="text-error mt-1 text-sm">
                {fieldErrors.confirmPassword}
              </p>
            )}
          </div>

          <Button
            type="submit"
            className="login__submit"
            disabled={loading}
            data-testid="set-password-submit"
          >
            {loading ? t(`${ns}.submitting`) : t(`${ns}.submit`)}
          </Button>
        </form>
      </div>
    </>
  )
}
