// 此表单提交用户名或邮箱与密码，成功后恢复受保护页面记录的返回位置。
import { BrandingAsset } from '@/features/branding/BrandingAsset'
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { login } from '@/shared/lib/api/auth'
import { mapApiError } from '@/shared/lib/api/errors'
import { useBrandingStore } from '@/features/branding/state/branding'
import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Navigate, useLocation, useNavigate } from 'react-router'
import { toast } from 'sonner'

// getRedirectTo 只接受状态对象中的斜线开头字符串，并排除登录页及其查询或片段形式。
// 此检查不解析 URL，也不规范化内部路径段，结果交给 React Router 导航。
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

// LoginForm 使用会话存储判断已登录状态，邮件开关只控制找回密码链接的可见性。
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

  // 已有用户时直接替换导航到有效返回位置，没有返回位置才使用首页。
  if (user) {
    return <Navigate to={redirectTo || '/'} replace />
  }

  const handleCredentialsSubmit = async (e: React.FormEvent) => {
    // API 登录函数负责应用会话，表单负责加载标记、错误提示与成功后的路由替换。
    e.preventDefault()
    setLoading(true)

    try {
      // 输入按当前文本直接提交，组件不裁剪用户名或密码，服务端决定凭据是否有效。
      await login(identifier, password)
      // 会话已由 login 写入存储，导航替换当前历史项以免后退回到登录表单。
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
      <BrandingAsset
        kind="logo"
        file={logoFile}
        version={logoVersion}
        alt={siteName}
        className="login__logo-image"
      />{' '}
      {siteName}
    </h1>
  )

  return (
    <>
      <title>{t('login.pageTitle', { siteName })}</title>
      <div className="login">
        <form onSubmit={handleCredentialsSubmit} className="login__form">
          {logoHeader}

          {/* 浏览器要求两个字段非空，输入状态分别保存账号标识与密码，不写入会话缓存。 */}
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

          {/* 邮件未启用时隐藏找回入口，账号密码登录本身不依赖邮件开关。 */}
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
