// All controls on this page update the same instance-wide preferences.
import { useTranslation } from 'react-i18next'
import { PreferencesPanel } from './account/PreferencesPanel'
export default function PreferencesSettings() {
  const { t } = useTranslation('settings')
  return (
    <div className="settings">
      <h1 className="settings__title">{t('nav.preferences')}</h1>
      <div className="settings__section">
        <PreferencesPanel />
      </div>
    </div>
  )
}
