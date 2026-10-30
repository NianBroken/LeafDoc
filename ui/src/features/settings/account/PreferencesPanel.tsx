import { toast } from 'sonner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/components/ui/select'
import { getAvailableLanguages } from '@/shared/lib/i18n'
import { useUserSettingsStore } from '@/shared/state/userSettings'
import { useTranslation } from 'react-i18next'

const DATE_FORMAT_OPTIONS = [
  { value: 'iso', labelKey: 'account.preferences.dateFormat.iso' },
  { value: 'mdy_slash', labelKey: 'account.preferences.dateFormat.mdySlash' },
  { value: 'dmy_slash', labelKey: 'account.preferences.dateFormat.dmySlash' },
  { value: 'dmy_dot', labelKey: 'account.preferences.dateFormat.dmyDot' },
] as const

const TIME_FORMAT_OPTIONS = [
  { value: '24h', labelKey: 'account.preferences.timeFormat.h24' },
  { value: '12h', labelKey: 'account.preferences.timeFormat.h12' },
] as const

export function PreferencesPanel() {
  const { t } = useTranslation('settings')
  const language = useUserSettingsStore((s) => s.language)
  const setLanguage = useUserSettingsStore((s) => s.setLanguage)
  const dateFormat = useUserSettingsStore((s) => s.dateFormat)
  const setDateFormat = useUserSettingsStore((s) => s.setDateFormat)
  const timeFormat = useUserSettingsStore((s) => s.timeFormat)
  const setTimeFormat = useUserSettingsStore((s) => s.setTimeFormat)

  return (
    <div data-testid="preferences-panel">
      <div className="settings__field">
        <label
          id="preferences-language-label"
          className="settings__field-label"
        >
          {t('account.preferences.languageLabel')}
        </label>
        <Select
          value={language}
          onValueChange={(value) => {
            void setLanguage(value).catch(() =>
              toast.error(t('account.preferences.languageChangeError')),
            )
          }}
        >
          <SelectTrigger
            data-testid="preferences-language-select"
            aria-labelledby="preferences-language-label"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {getAvailableLanguages().map((lang) => (
              <SelectItem key={lang.code} value={lang.code}>
                {lang.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="settings__field">
        <label
          id="preferences-dateformat-label"
          className="settings__field-label"
        >
          {t('account.preferences.dateFormatLabel')}
        </label>
        <Select
          value={dateFormat}
          onValueChange={(value) => {
            void setDateFormat(value).catch(() =>
              toast.error(t('account.preferences.dateFormatChangeError')),
            )
          }}
        >
          <SelectTrigger
            data-testid="preferences-dateformat-select"
            aria-labelledby="preferences-dateformat-label"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {DATE_FORMAT_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {t(option.labelKey)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="settings__field">
        <label
          id="preferences-timeformat-label"
          className="settings__field-label"
        >
          {t('account.preferences.timeFormatLabel')}
        </label>
        <Select
          value={timeFormat}
          onValueChange={(value) => {
            void setTimeFormat(value).catch(() =>
              toast.error(t('account.preferences.timeFormatChangeError')),
            )
          }}
        >
          <SelectTrigger
            data-testid="preferences-timeformat-select"
            aria-labelledby="preferences-timeformat-label"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {TIME_FORMAT_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {t(option.labelKey)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
