// 本文件读取标签建议和计数，查询值通过 URLSearchParams 编码。
import { fetchWithAuth } from '@/shared/lib/api/client'

// 标签文本与计数保持服务端返回形式，本层不重新统计页面。
export type TagCount = {
  tag: string
  count: number
}

// limit 始终发送，非空 filter 使用 q，selected 按重复参数保留顺序发送。
// 不去重、不去空白，也不在请求前验证数量上限。
export async function fetchTags(
  filter = '',
  limit = 50,
  selected: string[] = [],
): Promise<TagCount[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (filter) params.set('q', filter)
  for (const tag of selected) {
    params.append('selected', tag)
  }
  return await fetchWithAuth<TagCount[]>(`/api/tags?${params}`)
}
