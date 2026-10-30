// Error mapping preserves actual field and transport failures across all callers.
import { describe, expect, it } from 'vitest'
import i18next from '@/shared/lib/i18n'
import { ApiLocalizedError, mapApiError } from './errors'

describe('mapApiError', () => {
  it('returns specific field messages and their input keys', async () => {
    await i18next.changeLanguage('zh')
    const result = mapApiError(
      {
        error: 'validation_error',
        fields: [{ field: 'siteName', message: 'Site name must not be empty' }],
      },
      '保存失败',
    )
    expect(result.message).toBe('站点名称不能为空')
    expect(result.fields).toEqual({ siteName: '站点名称不能为空' })
  })
  it('uses a real localized reason when a template is missing', () => {
    expect(
      mapApiError(
        new ApiLocalizedError({
          code: 'test',
          template: 'not-in-catalog',
          message: '确切原因',
        }),
        '失败',
      ).message,
    ).toBe('确切原因')
  })
  it('distinguishes network failures from validation and permission errors', async () => {
    await i18next.changeLanguage('zh')
    expect(
      mapApiError(new TypeError('Failed to fetch'), '失败').message,
    ).toContain('网络连接失败')
  })
})
