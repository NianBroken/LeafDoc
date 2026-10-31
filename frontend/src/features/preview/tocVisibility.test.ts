// 表驱动覆盖空目录和三种模式，auto 同时检查三项与四项之间的展开边界。
import { describe, expect, it } from 'vitest'
import { isTocInitiallyCollapsed } from '@/features/preview/tocVisibility'

describe('isTocInitiallyCollapsed', () => {
  it.each([
    [0, 'always', true],
    [1, 'always', false],
    [4, 'always', false],
    [1, 'auto', true],
    [3, 'auto', true],
    [4, 'auto', false],
    [1, 'never', true],
    [4, 'never', true],
  ])('entryCount=%i mode=%s -> %s', (entryCount, mode, expected) => {
    expect(isTocInitiallyCollapsed(entryCount as number, mode as string)).toBe(
      expected,
    )
  })
})
