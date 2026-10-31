// 使用接口替身验证收藏加载、提前更新与失败恢复，通过可手动完成的 Promise 观察请求中状态。
import { expectDiagnostics } from '@/shared/testutil/expectedDiagnostics'
import type { Page } from '@/shared/lib/api/pages'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/lib/api/favorites', () => ({
  getFavorites: vi.fn(),
  addFavorite: vi.fn(),
  removeFavorite: vi.fn(),
}))

import * as favoritesAPI from '@/shared/lib/api/favorites'
import { useFavoritesStore } from '@/features/favorites/state/favorites'

function makePage(id: string): Page {
  // 加载动作只使用返回页面的 ID，其余字段提供符合接口类型的最小页面数据。
  return {
    id,
    slug: id,
    path: id,
    title: id,
    content: '',
    version: 'v1',
    kind: 'page',
  }
}

describe('useFavoritesStore', () => {
  beforeEach(() => {
    // 重设内存集合、加载标记及各接口替身，避免异步结果和调用记录串入其他用例。
    useFavoritesStore.setState({ favoritePageIds: new Set(), loaded: false })
    vi.mocked(favoritesAPI.getFavorites).mockReset()
    vi.mocked(favoritesAPI.addFavorite).mockReset()
    vi.mocked(favoritesAPI.removeFavorite).mockReset()
  })

  it('loadFavorites populates favoritePageIds from the API', async () => {
    vi.mocked(favoritesAPI.getFavorites).mockResolvedValue([
      makePage('page-1'),
      makePage('page-2'),
    ])

    await useFavoritesStore.getState().loadFavorites()

    expect(useFavoritesStore.getState().favoritePageIds).toEqual(
      new Set(['page-1', 'page-2']),
    )
    expect(useFavoritesStore.getState().loaded).toBe(true)
  })

  it('loadFavorites swallows API errors and leaves state as-is', async () => {
    // 显式登记预期告警，再验证加载动作正常返回且未填入收藏 ID。
    expectDiagnostics('warn', [
      ['Failed to load favorites:', new Error('network error')],
    ])
    vi.mocked(favoritesAPI.getFavorites).mockRejectedValue(
      new Error('network error'),
    )

    await expect(
      useFavoritesStore.getState().loadFavorites(),
    ).resolves.toBeUndefined()
    expect(useFavoritesStore.getState().favoritePageIds.size).toBe(0)
  })

  it('addFavorite optimistically adds the page id before the API resolves', async () => {
    // 暂不完成接口 Promise，先检查集合，再手动完成请求并确认状态保留。
    let resolveAdd: () => void = () => {}
    vi.mocked(favoritesAPI.addFavorite).mockReturnValue(
      new Promise((resolve) => {
        resolveAdd = () => resolve(undefined)
      }),
    )

    const promise = useFavoritesStore.getState().addFavorite('page-1')
    expect(useFavoritesStore.getState().favoritePageIds.has('page-1')).toBe(
      true,
    )

    resolveAdd()
    await promise
    expect(useFavoritesStore.getState().favoritePageIds.has('page-1')).toBe(
      true,
    )
  })

  it('addFavorite rolls back the optimistic update when the API call fails', async () => {
    // 拒绝结果必须向调用者抛出，同时将提前加入的 ID 从集合恢复掉。
    vi.mocked(favoritesAPI.addFavorite).mockRejectedValue(new Error('boom'))

    await expect(
      useFavoritesStore.getState().addFavorite('page-1'),
    ).rejects.toThrow('boom')
    expect(useFavoritesStore.getState().favoritePageIds.has('page-1')).toBe(
      false,
    )
  })

  it('removeFavorite optimistically removes the page id before the API resolves', async () => {
    // 预置收藏后延迟接口完成，分别检查请求进行中和完成后的移除结果。
    useFavoritesStore.setState({ favoritePageIds: new Set(['page-1']) })
    let resolveRemove: () => void = () => {}
    vi.mocked(favoritesAPI.removeFavorite).mockReturnValue(
      new Promise((resolve) => {
        resolveRemove = () => resolve(undefined)
      }),
    )

    const promise = useFavoritesStore.getState().removeFavorite('page-1')
    expect(useFavoritesStore.getState().favoritePageIds.has('page-1')).toBe(
      false,
    )

    resolveRemove()
    await promise
    expect(useFavoritesStore.getState().favoritePageIds.has('page-1')).toBe(
      false,
    )
  })

  it('removeFavorite rolls back the optimistic update when the API call fails', async () => {
    // 接口拒绝后原收藏应重新出现，调用者仍收到相同错误消息。
    useFavoritesStore.setState({ favoritePageIds: new Set(['page-1']) })
    vi.mocked(favoritesAPI.removeFavorite).mockRejectedValue(new Error('boom'))

    await expect(
      useFavoritesStore.getState().removeFavorite('page-1'),
    ).rejects.toThrow('boom')
    expect(useFavoritesStore.getState().favoritePageIds.has('page-1')).toBe(
      true,
    )
  })

  it('clearFavorites resets to empty and not-loaded', () => {
    // 从已加载且有收藏的状态调用清空，同时检查集合与加载标记。
    useFavoritesStore.setState({
      favoritePageIds: new Set(['page-1']),
      loaded: true,
    })

    useFavoritesStore.getState().clearFavorites()

    expect(useFavoritesStore.getState().favoritePageIds.size).toBe(0)
    expect(useFavoritesStore.getState().loaded).toBe(false)
  })
})
