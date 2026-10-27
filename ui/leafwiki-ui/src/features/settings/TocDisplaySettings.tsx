// Administrators choose only the default; visitors can always toggle the current document.
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { fetchWithAuth } from '@/lib/api/auth'
import { useConfigStore } from '@/stores/config'
export default function TocDisplaySettings() {
  const { t } = useTranslation('tocDisplay')
  const mode = useConfigStore((s) => s.tocMode)
  const [saving, setSaving] = useState(false)
  async function apply(value: string) {
    setSaving(true)
    try {
      await fetchWithAuth(`/api/admin/settings/toc-display`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode: value }),
      })
      await useConfigStore.getState().loadConfig()
      toast.success(t('savedToast'))
    } catch {
      toast.error(t('updateError'))
    } finally {
      setSaving(false)
    }
  }
  return (
    <div className="settings" data-testid="toc-display-settings">
      <h1 className="settings__title">{t('pageTitle')}</h1>
      <fieldset disabled={saving} className="space-y-4">
        {['auto', 'always', 'never'].map((value) => (
          <label key={value} className="flex items-center gap-3">
            <input
              type="radio"
              name="toc-mode"
              value={value}
              checked={mode === value}
              onChange={() => void apply(value)}
            />
            {t(`modes.${value}`)}
          </label>
        ))}
      </fieldset>
    </div>
  )
}
