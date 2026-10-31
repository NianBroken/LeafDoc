// 此页面只显示强制改密标题与当前账号改密面板，路由层决定何时使用该页面。
import { ChangeOwnPasswordPanel } from '@/features/settings/account/ChangeOwnPasswordPanel'
import { useTranslation } from 'react-i18next'
// 复用账号设置中的面板，避免在独立页面重复实现密码提交和会话更新。
export default function ForcePasswordChange() {
  const { t } = useTranslation('settings')
  return (
    <main className="mx-auto max-w-lg p-8">
      <h1 className="mb-6 text-2xl">{t('account.forcePasswordChange')}</h1>
      <ChangeOwnPasswordPanel />
    </main>
  )
}
