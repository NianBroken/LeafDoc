// 日期显示测试使用本地时间构造输入后转换为 ISO 文本，避免固定 UTC 时刻造成时区偏移。
// 相对时间用当前时间减四小时，检查中文单位而不固定日期库给出的整段提示。
import { describe, it, expect } from 'vitest'
import i18next from '@/shared/lib/i18n'
import {
  formatDateOnly,
  formatTimeOnly,
  formatRelativeTime,
} from '@/shared/lib/formatDate'
import { bcp47 } from '@/shared/lib/dateLocale'
describe('localized date formatting', () => {
  it('uses ISO date and 24-hour time by default', () => {
    // 未显式指定偏好时读取默认状态，分别断言日期排列与二十四小时制。
    const value = new Date(2026, 7, 27, 14, 30).toISOString()
    expect(formatDateOnly(value)).toBe('2026-08-27')
    expect(formatTimeOnly(value)).toBe('14:30')
  })
  it('honors explicit date formats', () => {
    // 显式传入日月年点分隔模式，确认它覆盖默认日期排列。
    const value = new Date(2026, 7, 27, 14, 30).toISOString()
    expect(
      formatDateOnly(value, 'zh', { dateFormat: 'dmy_dot', timeFormat: '24h' }),
    ).toBe('27.08.2026')
  })
  it('renders Chinese relative time and day periods', async () => {
    // 等待界面语言切换后检查相对时间和十二小时制的时段文字。
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
    // 这里核对中文、法语和西班牙语的明确区域标签，不遍历全部语言代码。
    expect(bcp47('zh')).toBe('zh-CN')
    expect(bcp47('fr')).toBe('fr-FR')
    expect(bcp47('es')).toBe('es-ES')
  })
})
