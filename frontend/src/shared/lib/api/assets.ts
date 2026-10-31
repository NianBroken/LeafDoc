// 本文件封装页面附件请求，并为附件面板维护进程内列表缓存和进行中的请求。
import { fetchWithAuth } from '@/shared/lib/api/client'

// 上传响应中的 file 由服务端提供，本层不根据本地文件名自行推断最终地址。
export type UploadAssetResponse = {
  file: string
}

// 面板展示项保存末段名称和规范后的附件地址。
export type PageAttachment = {
  name: string
  url: string
}

// 仅按 assets 前缀补齐根路径，其他输入都视为附件路径，不识别外部 URL。
function normalizeAssetUrl(path: string): string {
  if (path.startsWith('/assets/')) return path
  if (path.startsWith('assets/')) return `/${path}`
  return `/assets/${path}`
}

// 按规范后的 URL 去重并保留首次出现顺序，名称取最后一个斜线后的原始文本。
// 不解码名称，也不移除 URL 中的查询串或片段。
export function listPageAttachments(files: string[]): PageAttachment[] {
  const seen = new Set<string>()
  const attachments: PageAttachment[] = []
  for (const file of files) {
    const url = normalizeAssetUrl(file)
    if (seen.has(url)) continue
    seen.add(url)
    attachments.push({ name: url.split('/').pop() ?? url, url })
  }
  return attachments
}

// 两个模块级映射分别保存已完成列表和进行中的请求，按页面 ID 复用。
// 缓存不设置过期时间，也不在这里监听退出登录或其他客户端的附件变化。
const attachmentCache = new Map<string, PageAttachment[]>()
const attachmentRequests = new Map<string, Promise<PageAttachment[]>>()

/** 删除页面缓存与请求引用，不中止已发请求；旧请求完成后仍能再次写入缓存。 */
export function invalidatePageAttachments(pageId: string): void {
  attachmentCache.delete(pageId)
  attachmentRequests.delete(pageId)
}

// 优先返回缓存数组，其次等待同页进行中的 Promise，否则发起一次列表请求。
// 返回数组不复制，调用方修改会影响缓存；请求完成后清除该页进行中引用。
export async function getPageAttachments(
  pageId: string,
): Promise<PageAttachment[]> {
  const cached = attachmentCache.get(pageId)
  if (cached) return cached

  const inFlight = attachmentRequests.get(pageId)
  if (inFlight) return inFlight

  const request = getAssets(pageId)
    .then((files) => {
      const attachments = listPageAttachments(files)
      attachmentCache.set(pageId, attachments)
      return attachments
    })
    .finally(() => {
      attachmentRequests.delete(pageId)
    })

  attachmentRequests.set(pageId, request)
  return request
}

// 使用 file 表单字段上传，成功响应后才失效本页附件缓存，失败保留既有缓存。
export async function uploadAsset(
  pageId: string,
  file: File,
): Promise<UploadAssetResponse> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetchWithAuth<UploadAssetResponse>(
    `/api/pages/${pageId}/assets`,
    {
      method: 'POST',
      body: form,
    },
  )
  invalidatePageAttachments(pageId)
  return res
}

// 直接读取 files 字段，不走附件面板缓存，也不校验响应是否确为字符串数组。
export async function getAssets(pageId: string): Promise<string[]> {
  const typedData = await fetchWithAuth<{ files: string[] }>(
    `/api/pages/${pageId}/assets`,
    {},
  )
  return typedData.files
}

// 文件名作为单个路径片段编码，删除成功后失效列表，不在本地先行删项。
export async function deleteAsset(pageId: string, filename: string) {
  const res = await fetchWithAuth(
    `/api/pages/${pageId}/assets/${encodeURIComponent(filename)}`,
    {
      method: 'DELETE',
    },
  )
  invalidatePageAttachments(pageId)
  return res
}

// 新旧名称使用服务端要求的下划线字段名提交，成功后让下次读取重新获取列表。
export async function renameAsset(
  pageId: string,
  oldFilename: string,
  newFilename: string,
) {
  const res = await fetchWithAuth(`/api/pages/${pageId}/assets/rename`, {
    method: 'PUT',
    body: JSON.stringify({
      old_filename: oldFilename,
      new_filename: newFilename,
    }),
  })
  invalidatePageAttachments(pageId)
  return res
}
