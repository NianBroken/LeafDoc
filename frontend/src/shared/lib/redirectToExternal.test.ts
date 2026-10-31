// 替换 location 为可写对象后检查生成的地址，测试不让浏览器实际离开页面。
import { afterEach, describe, expect, it } from 'vitest'
import { redirectToExternal } from '@/shared/lib/redirectToExternal'

describe('redirectToExternal', () => {
  const originalLocation = window.location

  afterEach(() => {
    // 每例结束恢复原始 location，避免地址替身影响其他测试。
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    })
  })

  it('preserves the configured query and hash when adding the return URL', () => {
    // 绝对目标使用 URL 参数接口覆盖已有 redirect_uri，保留租户参数和目标片段。
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, href: '' },
    })

    redirectToExternal(
      'https://idp.example.com/login?tenant=a&redirect_uri=https%3A%2F%2Fold.example.com#section',
      '/protected?tab=x#anchor',
    )

    const redirected = new URL(window.location.href)
    expect(redirected.searchParams.get('tenant')).toBe('a')
    expect(redirected.searchParams.getAll('redirect_uri')).toHaveLength(1)
    expect(redirected.searchParams.get('redirect_uri')).toBe(
      `${originalLocation.origin}/protected?tab=x#anchor`,
    )
    expect(redirected.hash).toBe('#section')
  })

  it('preserves query and hash placement for a legacy relative URL', () => {
    // 相对目标走文本拼接路径，返回地址必须进入查询串而非落在地址片段之后。
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, href: '' },
    })

    redirectToExternal('/login?tenant=a#section', '/protected?tab=x#anchor')

    expect(window.location.href.indexOf('?')).toBeLessThan(
      window.location.href.indexOf('#'),
    )
    const redirected = new URL(window.location.href, originalLocation.origin)
    expect(redirected.origin).toBe(originalLocation.origin)
    expect(redirected.searchParams.get('tenant')).toBe('a')
    expect(redirected.searchParams.get('redirect_uri')).toBe(
      `${originalLocation.origin}/protected?tab=x#anchor`,
    )
    expect(redirected.hash).toBe('#section')
  })
})
