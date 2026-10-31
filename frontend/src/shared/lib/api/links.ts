// 本文件定义页面链接统计响应并封装查询，不在客户端重新解析 Markdown 链接。
import i18next from '@/shared/lib/i18n'
import { fetchWithAuth } from '@/shared/lib/api/client'

// 反向链接保存来源页面及目标 ID，broken 表示服务端判断的链接状态。
export type Backlink = {
  from_page_id: string
  from_path: string
  to_page_id: string
  from_title: string
  broken: boolean
}

// 出站链接保存目标路径、标题和目标 ID，无法解析时的具体空值由服务端返回。
export type OutgoingLink = {
  from_page_id: string
  to_page_id: string
  to_path: string
  to_page_title: string
  broken: boolean
}

// 四组计数与详细列表同时返回，本层不重新计数覆盖服务端结果。
export type LinkStatusCounts = {
  backlinks: number
  broken_incoming: number
  outgoings: number
  broken_outgoings: number
}

// 链接面板按入站、失效入站、出站、失效出站分别消费这些列表。
export type LinkStatusResult = {
  backlinks: Backlink[]
  broken_incoming: Backlink[]
  outgoings: OutgoingLink[]
  broken_outgoings: OutgoingLink[]
  counts: LinkStatusCounts
}

// 空页面 ID 在发请求前抛出当前语言提示，其余 ID 直接拼接到路径，不编码或去空白。
export async function fetchLinkStatus(
  pageId: string,
): Promise<LinkStatusResult> {
  if (!pageId)
    throw new Error(i18next.t('backlinks.pageIdRequired', { ns: 'viewer' }))
  return await fetchWithAuth<LinkStatusResult>(`/api/pages/${pageId}/links`)
}

// 全局失效链接列表提供来源定位与目标路径，不要求存在目标页面 ID。
export type BrokenLink = {
  from_page_id: string
  from_path: string
  from_title: string

  to_path: string
}

// 服务端以 links 属性包装列表。
export type BrokenLinksResult = {
  links: BrokenLink[]
}

// 请求全局失效链接结果，不进行页面过滤、排序或本地缓存。
export async function fetchBrokenLinks(): Promise<BrokenLinksResult> {
  return await fetchWithAuth<BrokenLinksResult>('/api/links/broken')
}
