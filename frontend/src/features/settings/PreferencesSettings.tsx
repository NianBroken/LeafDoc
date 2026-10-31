// 偏好设置页展示语言、日期和时间格式面板，控件通过共享状态保存实例级偏好。
import { useTranslation } from 'react-i18next'
import { PreferencesPanel } from '@/features/settings/account/PreferencesPanel'
// 页面只提供翻译标题和布局，提交请求与失败提示由 PreferencesPanel 负责。
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
