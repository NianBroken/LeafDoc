// 动态模块错误分类只检查提示文本，测试不发起模块下载，也不模拟网络恢复。
import { describe, expect, it } from 'vitest'
import { isDynamicImportChunkError } from '@/shared/lib/dynamicImportError'

describe('isDynamicImportChunkError', () => {
  it('matches the Chromium/V8 message (reproduced from a live redeploy)', () => {
    // 地址只是错误文本的一部分，匹配依据是浏览器给出的动态导入失败短语。
    expect(
      isDynamicImportChunkError(
        'Failed to fetch dynamically imported module: https://stuart.cloud.leafdoc.com/static/flowDiagram-HODETNUW-DScmEErP.js',
      ),
    ).toBe(true)
  })

  it('matches the Firefox message', () => {
    // Firefox 使用另一种失败短语，同样进入动态模块加载错误分类。
    expect(
      isDynamicImportChunkError(
        'error loading dynamically imported module: https://example.com/static/flowDiagram-abc.js',
      ),
    ).toBe(true)
  })

  it('matches the Safari/WebKit message', () => {
    // Safari 文本不带资源地址，仍由模块脚本导入失败短语命中。
    expect(isDynamicImportChunkError('Importing a module script failed.')).toBe(
      true,
    )
  })

  it('does not match an unrelated Mermaid parse error', () => {
    // 图表语法错误属于内容解析失败，不能归入资源下载失败。
    expect(
      isDynamicImportChunkError('Parse error on line 1: Expecting...'),
    ).toBe(false)
  })

  it('does not match a generic empty or unrelated message', () => {
    // 空文本与缺少 SVG 的提示均不命中三种浏览器短语。
    expect(isDynamicImportChunkError('')).toBe(false)
    expect(isDynamicImportChunkError('No SVG element in Mermaid output')).toBe(
      false,
    )
  })
})
