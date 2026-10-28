import { fetchWithAuth } from '@/shared/lib/api/client'

export type TagCount = {
  tag: string
  count: number
}

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
