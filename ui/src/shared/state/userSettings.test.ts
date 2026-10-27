import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useUserSettingsStore,
  DEFAULT_PREFERENCES,
} from '@/shared/state/userSettings'
import { fetchWithAuth } from '@/shared/lib/api/client'
vi.mock('@/shared/lib/api/client', () => ({ fetchWithAuth: vi.fn() }))
describe('global preferences', () => {
  beforeEach(() => {
    useUserSettingsStore.setState({ ...DEFAULT_PREFERENCES, loaded: false })
    vi.clearAllMocks()
  })
  it('defaults to simplified Chinese and explicit date/time formats', () => {
    expect(useUserSettingsStore.getState()).toMatchObject({
      language: 'zh',
      dateFormat: 'iso',
      timeFormat: '24h',
      autoSave: true,
    })
  })
  it('writes global preferences and applies the authoritative response', async () => {
    vi.mocked(fetchWithAuth).mockResolvedValue({
      ...DEFAULT_PREFERENCES,
      language: 'de',
    })
    await useUserSettingsStore.getState().setLanguage('de')
    expect(fetchWithAuth).toHaveBeenCalledWith(
      '/api/admin/settings/preferences',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ language: 'de' }),
      }),
    )
    expect(useUserSettingsStore.getState().language).toBe('de')
  })
  it('keeps committed state when saving fails', async () => {
    vi.mocked(fetchWithAuth).mockRejectedValue(new Error('unavailable'))
    await expect(
      useUserSettingsStore.getState().setDateFormat('dmy_dot'),
    ).rejects.toThrow()
    expect(useUserSettingsStore.getState().dateFormat).toBe('iso')
  })
})
