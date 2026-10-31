// 标题键依据标题数和下载数选择，不读取实际标题文字或附件文件。
import { describe, expect, it } from 'vitest'
import { tocTitleKey } from '@/features/preview/tocTitleKey'

describe('tocTitleKey', () => {
  // 存在标题时，无论是否同时存在附件，都使用页内目录名称。
  it('prefers "on this page" when there are headings', () => {
    expect(tocTitleKey(3, 0)).toBe('toc.onThisPage')
    expect(tocTitleKey(3, 2)).toBe('toc.onThisPage')
  })

  // 没有标题但存在附件时，将同一面板命名为下载列表。
  it('falls back to "downloads" when there are only attachments', () => {
    expect(tocTitleKey(0, 2)).toBe('toc.downloads')
  })

  // 两类条目都为空仍有确定的标题键，不返回空值。
  it('falls back to "on this page" when neither headings nor attachments exist', () => {
    expect(tocTitleKey(0, 0)).toBe('toc.onThisPage')
  })
})
