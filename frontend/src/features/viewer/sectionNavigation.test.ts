// Empty-section navigation follows the supplied tree order including nested sections.
import { describe, expect, it } from 'vitest'
import { firstDescendantPage } from './sectionNavigation'
import {
  NODE_KIND_PAGE,
  NODE_KIND_SECTION,
  type PageNode,
} from '@/shared/lib/api/pages'

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
  it('uses directory order rather than title order', () => {
    expect(firstDescendantPage(node('root', [node('z'), node('a')]))?.id).toBe(
      'z',
    )
  })
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
  it('returns no target when all descendants are empty sections', () => {
    expect(
      firstDescendantPage(node('root', [node('nested', [node('empty', [])])])),
    ).toBeNull()
  })
})
