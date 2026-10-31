// 页面路径测试直接比较工具函数返回值，覆盖查询键、路由转换、相对链接和删除后跳转目标。
// 测试不访问路由器或页面树，路径存在性和删除操作本身不在验证范围内。
import { describe, expect, it } from 'vitest'
import {
  getDeleteRedirectRoutePath,
  getParentWikiRoutePath,
  getWikiTargetRoutePath,
  normalizeWikiRoutePath,
  resolveWikiLinkPath,
  toWikiLookupPath,
} from '@/shared/lib/wikiPath'

describe('normalizeWikiRoutePath', () => {
  it('adds a leading slash and strips trailing slashes', () => {
    // 分别使用缺少前导斜线和带末尾斜线的输入，核对相同规范路径。
    expect(normalizeWikiRoutePath('docs/guide')).toBe('/docs/guide')
    expect(normalizeWikiRoutePath('/docs/guide/')).toBe('/docs/guide')
  })

  it('strips query and hash fragments', () => {
    // 查找页面只使用路径，查询参数和地址片段均从结果排除。
    expect(normalizeWikiRoutePath('/docs/guide?q=1#section')).toBe(
      '/docs/guide',
    )
  })

  it('keeps root as /', () => {
    // 根路径不因移除末尾斜线而变为空字符串。
    expect(normalizeWikiRoutePath('/')).toBe('/')
  })
})

describe('toWikiLookupPath', () => {
  it('converts absolute wiki routes to tree lookup keys', () => {
    // 树查询键不带前导斜线，根节点使用空查询键。
    expect(toWikiLookupPath('/docs/getting-started')).toBe(
      'docs/getting-started',
    )
    expect(toWikiLookupPath('/')).toBe('')
  })
})

describe('getWikiTargetRoutePath', () => {
  it('maps edit and history routes back to the view path', () => {
    // 编辑与历史前缀去掉后仍保持页面路径，历史根指向站点根。
    expect(getWikiTargetRoutePath('/e/docs/guide')).toBe('/docs/guide')
    expect(getWikiTargetRoutePath('/history/docs/guide')).toBe('/docs/guide')
    expect(getWikiTargetRoutePath('/history')).toBe('/')
    expect(getWikiTargetRoutePath('/docs/guide')).toBe('/docs/guide')
  })
})

describe('resolveWikiLinkPath', () => {
  it('resolves relative links with page-as-folder semantics', () => {
    // 当前页面作为目录基准，普通子路径、上级目录和点目录按 URL 规则解析。
    expect(resolveWikiLinkPath('/docs/guide', 'setup')).toBe(
      '/docs/guide/setup',
    )
    expect(resolveWikiLinkPath('/docs/guide', '../other')).toBe('/docs/other')
    expect(resolveWikiLinkPath('/docs/guide', './child/page')).toBe(
      '/docs/guide/child/page',
    )
  })

  it('resolves absolute wiki paths from the site root', () => {
    // 根路径目标不接在当前页面目录之后。
    expect(resolveWikiLinkPath('/docs/guide', '/root-page')).toBe('/root-page')
  })
})

describe('getParentWikiRoutePath', () => {
  it('returns / for top-level pages', () => {
    // 顶层页面与根节点的父路径都归为根路径。
    expect(getParentWikiRoutePath('/docs')).toBe('/')
    expect(getParentWikiRoutePath('/')).toBe('/')
  })

  it('returns the parent for nested pages', () => {
    // 多层路径只去掉末尾片段，保留父级完整路径。
    expect(getParentWikiRoutePath('/docs/guide/setup')).toBe('/docs/guide')
  })
})

describe('getDeleteRedirectRoutePath', () => {
  it('redirects to the parent when the deleted page is open', () => {
    // 当前查看目标正是删除目标，跳转结果使用删除目标的父路径。
    expect(getDeleteRedirectRoutePath('/docs/guide', '/docs/guide')).toBe(
      '/docs',
    )
  })

  it('redirects from editor and history routes for the deleted page', () => {
    // 比较删除目标前先去掉工作模式前缀，避免编辑或历史页面停留在被删目标。
    expect(getDeleteRedirectRoutePath('/e/docs/guide', '/docs/guide')).toBe(
      '/docs',
    )
    expect(
      getDeleteRedirectRoutePath('/history/docs/guide', '/docs/guide'),
    ).toBe('/docs')
  })

  it('redirects to the parent when a nested route under the deleted page is open', () => {
    // 当前目标位于被删目录下时，返回被删目录的父级而非当前子页的父级。
    expect(getDeleteRedirectRoutePath('/docs/guide/setup', '/docs/guide')).toBe(
      '/docs',
    )
  })

  it('keeps the current route when another page is open', () => {
    // 无关页面不被删除操作带离，结果保持当前路径。
    expect(getDeleteRedirectRoutePath('/docs/other', '/docs/guide')).toBe(
      '/docs/other',
    )
  })
})
