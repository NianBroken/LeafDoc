// 保存从当前用户收藏接口取得的页面 ID，不写入本地存储。
// 加载动作每次调用都会请求接口，loaded 仅记录成功加载状态，不在此处阻止重复请求。
// 应用层在会话变化时安排加载或清空，添加与移除动作在请求完成前就更新集合。

import {
  addFavorite as addFavoriteAPI,
  getFavorites,
  removeFavorite as removeFavoriteAPI,
} from '@/shared/lib/api/favorites'
import { create } from 'zustand'

type FavoritesStore = {
  favoritePageIds: Set<string>
  loaded: boolean
  loadFavorites: () => Promise<void>
  addFavorite: (pageId: string) => Promise<void>
  removeFavorite: (pageId: string) => Promise<void>
  clearFavorites: () => void
}

export const useFavoritesStore = create<FavoritesStore>()((set, get) => ({
  favoritePageIds: new Set(),
  loaded: false,
  loadFavorites: async () => {
    // 用接口返回的页面 ID 整体替换集合；加载失败记录告警并保留已有集合及加载标记。
    try {
      const pages = await getFavorites()
      set({ favoritePageIds: new Set(pages.map((p) => p.id)), loaded: true })
    } catch (err) {
      console.warn('Failed to load favorites:', err)
    }
  },
  addFavorite: async (pageId: string) => {
    // 复制集合再加入目标 ID，使订阅者立即收到新引用，并保存原引用供失败恢复。
    const previous = get().favoritePageIds
    const optimistic = new Set(previous)
    optimistic.add(pageId)
    set({ favoritePageIds: optimistic })
    try {
      await addFavoriteAPI(pageId)
    } catch (err) {
      // 恢复本次动作开始时的完整集合，再抛出错误供按钮显示提示。
      set({ favoritePageIds: previous })
      throw err
    }
  },
  removeFavorite: async (pageId: string) => {
    // 先从集合副本移除目标 ID，请求失败时恢复动作开始时的集合。
    const previous = get().favoritePageIds
    const optimistic = new Set(previous)
    optimistic.delete(pageId)
    set({ favoritePageIds: optimistic })
    try {
      await removeFavoriteAPI(pageId)
    } catch (err) {
      set({ favoritePageIds: previous })
      throw err
    }
  },
  // 清空仅修改内存状态，不调用接口，也不取消已经发出的请求。
  clearFavorites: () => set({ favoritePageIds: new Set(), loaded: false }),
}))
