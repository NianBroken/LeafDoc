import { describe, it, expect } from 'vitest'
import i18next from './i18n'
import {
  formatDateOnly,
  formatTimeOnly,
  formatRelativeTime,
} from './formatDate'
import { bcp47 } from './dateLocale'
describe('localized date formatting', () => {
  it('uses ISO date and 24-hour time by default', () => {
    const value = new Date(2026, 7, 27, 14, 30).toISOString()
    expect(formatDateOnly(value)).toBe('2026-08-27')
    expect(formatTimeOnly(value)).toBe('14:30')
  })
  it('honors explicit date formats', () => {
    const value = new Date(2026, 7, 27, 14, 30).toISOString()
    expect(
      formatDateOnly(value, 'zh', { dateFormat: 'dmy_dot', timeFormat: '24h' }),
    ).toBe('27.08.2026')
  })
  it('renders Chinese relative time and day periods', async () => {
    await i18next.changeLanguage('zh')
    expect(
      formatRelativeTime(new Date(Date.now() - 4 * 3600000).toISOString()),
    ).toContain('小时')
    expect(
      formatRelativeTime(new Date(Date.now() - 4 * 3600000).toISOString()),
    ).not.toMatch(/hours|ago|about/)
    expect(
      formatTimeOnly(new Date(2026, 7, 27, 14, 30).toISOString(), 'zh', {
        dateFormat: 'iso',
        timeFormat: '12h',
      }),
    ).toContain('下午')
  })
  it('maps all shipped locales', () => {
    expect(bcp47('zh')).toBe('zh-CN')
    expect(bcp47('fr')).toBe('fr-FR')
    expect(bcp47('es')).toBe('es-ES')
  })
})
