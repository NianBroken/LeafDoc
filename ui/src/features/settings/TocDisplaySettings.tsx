import { mapApiError } from '@/shared/lib/api/errors'
// Administrators choose only the default; visitors can always toggle the current document.
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { fetchWithAuth } from '@/shared/lib/api/client'
import { useConfigStore } from '@/shared/state/config'
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
    } catch (err) {
      toast.error(mapApiError(err, t('updateError')).message)
    } finally {
      setSaving(false)
    }
  }
  return (
    <div className="settings" data-testid="toc-display-settings">
      <h1 className="settings__title">{t('pageTitle')}</h1>
      <section className="settings__section">
        <h2 className="settings__section-title">{t('sectionTitle')}</h2>
        <p className="settings__section-description">
          {t('sectionDescription')}
        </p>
        <fieldset disabled={saving} className="space-y-2">
          <legend className="sr-only">{t('sectionTitle')}</legend>
          {['auto', 'always', 'never'].map((value) => (
            <label
              key={value}
              className="border-surface-border hover:bg-surface-alt flex cursor-pointer items-start gap-3 rounded-md border p-4 transition-colors"
            >
              <input
                type="radio"
                name="toc-mode"
                value={value}
                checked={mode === value}
                onChange={() => void apply(value)}
                className="accent-brand mt-1"
              />
              <span className="flex flex-col gap-1">
                <span className="text-interface-text text-sm font-medium">
                  {t(`modes.${value}`)}
                </span>
                <span className="text-muted-foreground text-sm leading-5">
                  {t(`modeDescriptions.${value}`)}
                </span>
              </span>
            </label>
          ))}
        </fieldset>
      </section>
    </div>
  )
}
