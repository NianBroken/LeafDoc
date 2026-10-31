// 本文件使用具体错误样本核对界面提示，分别覆盖字段定位、缺失翻译和网络失败。
// 翻译实例与生产代码共用，测试显式选择中文后检查对应提示。
import { describe, expect, it } from 'vitest'
import i18next from '@/shared/lib/i18n'
import { ApiLocalizedError, mapApiError } from './errors'

describe('mapApiError', () => {
  // 服务端字段名 siteName 保持原样，消息通过 errors 命名空间转换为中文。
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
  // 词条不存在时返回详细错误自带的实际消息，而不是丢失原因后只显示通用失败。
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
  // 使用 Failed to fetch 类型错误触发网络提示分支，不把它解释为表单字段错误。
  it('distinguishes network failures from validation and permission errors', async () => {
    await i18next.changeLanguage('zh')
    expect(
      mapApiError(new TypeError('Failed to fetch'), '失败').message,
    ).toContain('网络连接失败')
  })
})
