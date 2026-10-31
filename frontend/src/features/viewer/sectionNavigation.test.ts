// 用内存节点核验后代导航顺序，覆盖原顺序优先、深层页面优先与完全空章节三种结果。
import { describe, expect, it } from 'vitest'
import { firstDescendantPage } from './sectionNavigation'
import {
  NODE_KIND_PAGE,
  NODE_KIND_SECTION,
  type PageNode,
} from '@/shared/lib/api/pages'

// children 为 null 的节点表示页面，任何数组都表示章节，包括没有子项的空数组。
function node(id: string, children: PageNode[] | null = null): PageNode {
  return {
    id,
    title: id,
    slug: id,
    path: id,
    version: id,
    kind: children === null ? NODE_KIND_PAGE : NODE_KIND_SECTION,
    children,
  }
}

describe('firstDescendantPage', () => {
  // 故意让标题 z 排在 a 前面，确认没有额外按标题排序。
  it('uses directory order rather than title order', () => {
    expect(firstDescendantPage(node('root', [node('z'), node('a')]))?.id).toBe(
      'z',
    )
  })
  // 跳过空章节后先进入嵌套章节，得到其中页面，再考虑后续兄弟节点。
  it('finds a nested page before a later sibling', () => {
    expect(
      firstDescendantPage(
        node('root', [
          node('empty', []),
          node('nested', [node('first')]),
          node('last'),
        ]),
      )?.id,
    ).toBe('first')
  })
  // 任意层级都没有页面时返回 null，不把空章节作为导航目标。
  it('returns no target when all descendants are empty sections', () => {
    expect(
      firstDescendantPage(node('root', [node('nested', [node('empty', [])])])),
    ).toBeNull()
  })
})
