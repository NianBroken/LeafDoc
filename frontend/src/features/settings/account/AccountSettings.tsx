// 当前账号设置组合头像和自身密码两个面板，表单提交分别由子组件负责。
import { useTranslation } from 'react-i18next'
import { useSetTitle } from '@/features/viewer/setTitle'
import { AvatarPanel } from '@/features/settings/account/AvatarPanel'
import { ChangeOwnPasswordPanel } from '@/features/settings/account/ChangeOwnPasswordPanel'

// 页面标题同步到浏览器标题，两个区块的标题和操作说明使用 settings 命名空间。
export default function AccountSettings() {
  const { t } = useTranslation('settings')

  useSetTitle({ title: t('account.pageTitle') })

  return (
    <div className="settings">
      <h1 className="settings__title">{t('account.pageTitle')}</h1>
      <div className="settings__section">
        <h2 className="settings__section-title">
          {t('account.avatar.sectionTitle')}
        </h2>
        <p className="settings__section-description">
          {t('account.avatar.sectionDescription')}
        </p>
        <AvatarPanel />
      </div>

      <div className="settings__section">
        <h2 className="settings__section-title">
          {t('account.passwordSectionTitle')}
        </h2>
        <p className="settings__section-description">
          {t('account.passwordSectionDescription')}
        </p>
        <ChangeOwnPasswordPanel />
      </div>
    </div>
  )
}
