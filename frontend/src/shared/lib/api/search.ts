// 本文件封装服务端全文搜索，不与本地页面树的名称匹配算法混用。
import type { PageKind } from '@/shared/contracts/page'
import { fetchWithAuth } from '@/shared/lib/api/client'

// 命中项包含服务端摘要、排序分值和标签，不包含完整页面正文。
export type SearchResultItem = {
  page_id: string
  path: string
  title: string
  kind: PageKind
  excerpt: string
  rank: number
  tags: string[]
}

// 标签分面保存标签及其计数，供结果界面继续筛选。
export type SearchTagFacet = {
  tag: string
  count: number
}

// 分页结果允许 items 为 null，计数、分页值与分面均采用服务端返回值。
export type SearchResult = {
  count: number
  items: SearchResultItem[] | null
  limit: number
  offset: number
  tag_facets: SearchTagFacet[]
}

// 负偏移设为零，大小超出一到一百时使用十，不进一步校验 NaN 或非整数。
// 无查询且无标签时直接返回固定空结果，其分页值不沿用调用方输入。
// 标签以重复 tags 参数发送，保留顺序与重复值，不在本层去空白或去重。
export async function searchPages(
  query: string,
  offset: number,
  limit: number,
  tags: string[] = [],
): Promise<SearchResult> {
  if (offset < 0) offset = 0
  if (limit < 1 || limit > 100) limit = 10

  if (!query && tags.length === 0) {
    return { count: 0, items: [], limit: 10, offset: 0, tag_facets: [] }
  }

  const params = new URLSearchParams({
    offset: String(offset),
    limit: String(limit),
  })
  if (query) params.set('q', query)
  for (const tag of tags) {
    params.append('tags', tag)
  }

  return await fetchWithAuth<SearchResult>(`/api/search?${params}`)
}
