// 本文件只封装收藏接口，不维护前端收藏缓存，页面状态由调用方刷新。
import { fetchWithAuth } from '@/shared/lib/api/client'
import type { Page } from '@/shared/lib/api/pages'

// 从响应的 pages 字段提取列表，不进行去重或排序。
export async function getFavorites(): Promise<Page[]> {
  const res = await fetchWithAuth<{ pages: Page[] }>('/api/favorites')
  return res.pages
}

// 对页面收藏子资源发送 PUT，等待完成后忽略返回正文。
export async function addFavorite(pageId: string): Promise<void> {
  await fetchWithAuth(`/api/pages/${pageId}/favorite`, { method: 'PUT' })
}

// 对同一子资源发送 DELETE，不在此判断页面是否已经收藏。
export async function removeFavorite(pageId: string): Promise<void> {
  await fetchWithAuth(`/api/pages/${pageId}/favorite`, { method: 'DELETE' })
}
