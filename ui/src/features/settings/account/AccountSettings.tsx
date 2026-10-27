import { useTranslation } from 'react-i18next'
import { useSetTitle } from '@/features/viewer/setTitle'
import { AvatarPanel } from '@/features/settings/account/AvatarPanel'
import { ChangeOwnPasswordPanel } from '@/features/settings/account/ChangeOwnPasswordPanel'

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
