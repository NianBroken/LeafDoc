// 本面板为实例偏好提供三个受控选择器，每次选择立即调用共享状态的异步保存动作。
// 请求失败经统一错误映射显示提示，字段当前值与保存结果由状态容器管理。
import { mapApiError } from '@/shared/lib/api/errors'
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

// 值是存储与接口使用的稳定格式标识，labelKey 只决定当前界面显示的译文。
const DATE_FORMAT_OPTIONS = [
  { value: 'iso', labelKey: 'account.preferences.dateFormat.iso' },
  { value: 'mdy_slash', labelKey: 'account.preferences.dateFormat.mdySlash' },
  { value: 'dmy_slash', labelKey: 'account.preferences.dateFormat.dmySlash' },
  { value: 'dmy_dot', labelKey: 'account.preferences.dateFormat.dmyDot' },
] as const

// 时间只提供二十四小时与十二小时两个选项，不在本组件内格式化实际日期。
const TIME_FORMAT_OPTIONS = [
  { value: '24h', labelKey: 'account.preferences.timeFormat.h24' },
  { value: '12h', labelKey: 'account.preferences.timeFormat.h12' },
] as const

// 单独订阅三个值与三个保存动作，控件不维护与共享状态重复的本地副本。
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
      {/* 可选语言来自共享语言目录，保存失败显示语言专属提示。标签 ID 与选择框可访问名称相连。 */}
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
            void setLanguage(value).catch((err) =>
              toast.error(
                mapApiError(err, t('account.preferences.languageChangeError'))
                  .message,
              ),
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

      {/* 日期格式逐项保存，展示名称随设置命名空间翻译，不向接口发送展示文字。 */}
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
            void setDateFormat(value).catch((err) =>
              toast.error(
                mapApiError(err, t('account.preferences.dateFormatChangeError'))
                  .message,
              ),
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

      {/* 时间格式保存与其他选择器相互独立，没有汇总提交按钮或批量保存动作。 */}
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
            void setTimeFormat(value).catch((err) =>
              toast.error(
                mapApiError(err, t('account.preferences.timeFormatChangeError'))
                  .message,
              ),
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
