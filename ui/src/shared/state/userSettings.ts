// Shared UI preferences are loaded for every visitor, independently of account sessions.
import { create } from 'zustand'
import { fetchWithAuth } from '@/shared/lib/api/client'
import i18next from '@/shared/lib/i18n'

export type GlobalPreferences = {
  language: string
  dateFormat: string
  timeFormat: string
}
export const DEFAULT_PREFERENCES: GlobalPreferences = {
  language: 'zh',
  dateFormat: 'iso',
  timeFormat: '24h',
}
type Store = GlobalPreferences & {
  loaded: boolean
  apply: (value: GlobalPreferences) => void
  setLanguage: (value: string) => Promise<void>
  setDateFormat: (value: string) => Promise<void>
  setTimeFormat: (value: string) => Promise<void>
}
// Writes are serialized; the service merges patches atomically and returns authoritative values.
let pending: Promise<unknown> = Promise.resolve()
function save(patch: Partial<GlobalPreferences>): Promise<void> {
  const task = pending
    .catch(() => undefined)
    .then(async () => {
      const response = await fetchWithAuth<GlobalPreferences>(
        `/api/admin/settings/preferences`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(patch),
        },
      )
      useUserSettingsStore.getState().apply(response)
    })
  pending = task
  return task
}
export const useUserSettingsStore = create<Store>((set) => ({
  ...DEFAULT_PREFERENCES,
  loaded: false,
  apply: (value) => {
    set({
      language: value.language,
      dateFormat: value.dateFormat,
      timeFormat: value.timeFormat,
      loaded: true,
    })
    void i18next.changeLanguage(value.language)
  },
  setLanguage: (language) => save({ language }),
  setDateFormat: (dateFormat) => save({ dateFormat }),
  setTimeFormat: (timeFormat) => save({ timeFormat }),
}))
