// 检查收藏 ID 在树索引中的解析、标题排序和缺失节点过滤，使用内存路由渲染链接。
import type { PageNode } from '@/shared/lib/api/pages'
import { useFavoritesStore } from '@/features/favorites/state/favorites'
import { useTreeStore } from '@/features/tree/state/tree'
import { render, screen } from '@testing-library/react'
import type React from 'react'
import { MemoryRouter } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FavoritesSection } from '@/features/favorites/FavoritesSection'

vi.mock('react-i18next', () => ({
  initReactI18next: {
    type: '3rdParty',
    init: () => {},
  },
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock('@/shared/components/TooltipWrapper', () => ({
  TooltipWrapper: ({ children }: { children: React.ReactNode }) => children,
}))

function makeNode(id: string, title: string): PageNode {
  // 创建无子节点的页面数据，路径直接使用 ID，测试只关注收藏列表的排列与显示。
  return {
    id,
    title,
    slug: id,
    path: id,
    version: 'v1',
    children: null,
    kind: 'page',
  }
}

describe('FavoritesSection', () => {
  beforeEach(() => {
    // 同时清空收藏集合与树索引，使空状态和缺失引用用例彼此独立。
    useFavoritesStore.setState({ favoritePageIds: new Set(), loaded: false })
    useTreeStore.setState({ byId: {} })
  })

  it('renders nothing when there are no favorites', () => {
    const { container } = render(
      <MemoryRouter>
        <FavoritesSection />
      </MemoryRouter>,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('renders favorited pages resolved from the tree store, sorted by title', () => {
    // 收藏顺序刻意与标题顺序相反，以元素排列断言组件确实执行标题排序。
    useTreeStore.setState({
      byId: {
        'page-1': makeNode('page-1', 'Zebra'),
        'page-2': makeNode('page-2', 'Alpha'),
      },
    })
    useFavoritesStore.setState({
      favoritePageIds: new Set(['page-1', 'page-2']),
    })

    render(
      <MemoryRouter>
        <FavoritesSection />
      </MemoryRouter>,
    )

    const items = screen.getAllByTestId('favorite-item')
    expect(items).toHaveLength(2)
    expect(items[0]).toHaveTextContent('Alpha')
    expect(items[1]).toHaveTextContent('Zebra')
  })

  it('silently skips a favorited page id that no longer resolves in the tree', () => {
    // 保留一个有效引用和一个失效引用，最终只应显示能够从树中取得的节点。
    useTreeStore.setState({
      byId: { 'page-1': makeNode('page-1', 'Still here') },
    })
    useFavoritesStore.setState({
      favoritePageIds: new Set(['page-1', 'stale-id']),
    })

    render(
      <MemoryRouter>
        <FavoritesSection />
      </MemoryRouter>,
    )

    expect(screen.getAllByTestId('favorite-item')).toHaveLength(1)
    expect(screen.getByText('Still here')).toBeInTheDocument()
  })
})
