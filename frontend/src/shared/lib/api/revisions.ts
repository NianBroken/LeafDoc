// 本文件定义历史版本快照与比较结果，并封装读取和恢复接口。
import { fetchWithAuth } from '@/shared/lib/api/client'

// 用户标签只含标识和显示名，不携带完整账号资料。
export type RevisionUserLabel = {
  id: string
  username: string
}

// 版本元信息用内容与附件清单摘要引用快照，正文在 RevisionSnapshot 中独立返回。
export type Revision = {
  id: string
  pageId: string
  parentId?: string
  type: string
  authorId: string
  author?: RevisionUserLabel
  createdAt: string
  title: string
  slug: string
  kind: string
  path: string
  contentHash: string
  assetManifestHash: string
  pageCreatedAt?: string
  pageUpdatedAt?: string
  creatorId?: string
  lastAuthorId?: string
  summary?: string
}

// 版本附件使用名称、SHA256 和字节数描述，MIME 类型可缺省。
export type RevisionAsset = {
  name: string
  sha256: string
  sizeBytes: number
  mimeType?: string
}

// 完整快照把版本信息、正文与附件清单组合，附件字节通过独立地址访问。
export type RevisionSnapshot = {
  revision: Revision
  content: string
  assets: RevisionAsset[]
}

// 单项附件差异标记新增、删除或修改，不在此结构携带差异字节。
export type RevisionAssetChange = {
  name: string
  status: 'added' | 'removed' | 'modified'
}

// 服务端返回两侧快照和差异标记，本层不重新比较哈希或正文。
export type RevisionComparison = {
  base: RevisionSnapshot
  target: RevisionSnapshot
  contentChanged: boolean
  assetChanges: RevisionAssetChange[]
}

// 游标分页由 nextCursor 继续定位，不在客户端把游标当页码解释。
export type RevisionListResponse = {
  revisions: Revision[]
  nextCursor: string
}

// 非空 cursor 才发送，limit 始终转成文本，不在此校验范围或整数性。
export async function listRevisions(
  pageId: string,
  cursor = '',
  limit = 50,
): Promise<RevisionListResponse> {
  const params = new URLSearchParams()
  if (cursor) params.set('cursor', cursor)
  params.set('limit', String(limit))
  const query = params.toString()
  return await fetchWithAuth<RevisionListResponse>(
    `/api/pages/${pageId}/revisions${query ? `?${query}` : ''}`,
  )
}

// 读取服务端认定的最新版本，不通过前端列表顺序推断。
export async function getLatestRevision(pageId: string): Promise<Revision> {
  return await fetchWithAuth<Revision>(`/api/pages/${pageId}/revisions/latest`)
}

// 页面 ID 和版本 ID 直接放入路径，返回正文与附件描述，不自动下载附件。
export async function getRevisionSnapshot(
  pageId: string,
  revisionId: string,
): Promise<RevisionSnapshot> {
  return await fetchWithAuth<RevisionSnapshot>(
    `/api/pages/${pageId}/revisions/${revisionId}`,
  )
}

// 两侧版本 ID 通过 URLSearchParams 编码到查询值，比较操作由服务端完成。
export async function compareRevisions(
  pageId: string,
  baseRevisionId: string,
  targetRevisionId: string,
): Promise<RevisionComparison> {
  const params = new URLSearchParams({
    base: baseRevisionId,
    target: targetRevisionId,
  })
  return await fetchWithAuth<RevisionComparison>(
    `/api/pages/${pageId}/revisions/compare?${params.toString()}`,
  )
}

// 请求将指定历史版本恢复为当前页面，不在此改写页面缓存或跳转路由。
export async function restoreRevision(pageId: string, revisionId: string) {
  return await fetchWithAuth(
    `/api/pages/${pageId}/revisions/${revisionId}/restore`,
    {
      method: 'POST',
    },
  )
}

// 按斜线拆分后移除空片段，逐段编码并保留层级，不把点目录另行规范化。
function encodeAssetName(name: string): string {
  return name
    .split('/')
    .filter(Boolean)
    .map((segment) => encodeURIComponent(segment))
    .join('/')
}

// 返回不含部署基础前缀的历史附件接口路径，调用方负责按使用场景处理前缀。
// 附件名去掉前导斜线后逐段编码，页面和版本 ID 仍原样拼接。
export function buildRevisionAssetUrl(
  pageId: string,
  revisionId: string,
  assetName: string,
): string {
  const normalizedAssetName = assetName.replace(/^\/+/, '')
  return `/api/pages/${pageId}/revisions/${revisionId}/assets/${encodeAssetName(
    normalizedAssetName,
  )}`
}
