// 本文件定义页面与树节点响应，并封装创建、更新、移动、重构和路径查询请求。
// 请求完成不在这里更新页面树或编辑器状态，缓存与界面刷新由调用方编排。
import type { PageKind } from '@/shared/contracts/page'
import { fetchWithAuth } from '@/shared/lib/api/client'

// 页面类型常量复用共享定义，避免接口封装再维护一套标识。
export { NODE_KIND_PAGE, NODE_KIND_SECTION } from '@/shared/contracts/page'

// 元信息同时保留作者 ID 和可选显示对象，时间按服务端字符串表示。
export type PageMetadata = {
  createdAt: string
  updatedAt: string
  creatorId: string
  lastAuthorId: string
  creator?: {
    id: string
    username: string
  }
  lastAuthor?: {
    id: string
    username: string
  }
}

// 树节点包含路径、版本与可空子列表，未携带页面 Markdown 正文。
export type PageNode = {
  id: string
  title: string
  slug: string
  path: string
  version: string
  parentId?: string | null
  children: PageNode[] | null
  kind: PageKind
  pinned?: boolean
  metadata?: PageMetadata // 元信息在类型中可缺省，消费者需处理不存在的状态。
}

// 页面详情保存正文、标签和字符串属性，不包含树节点的 children 列表。
export interface Page {
  id: string
  slug: string
  path: string
  title: string
  content: string
  tags?: string[]
  properties?: Record<string, string>
  version: string
  kind: PageKind
  metadata?: PageMetadata // 详情读取不能假定每个响应都含元信息。
}

// 永久链接解析结果通过稳定 ID 给出当前 slug 和页面路径。
export type PermalinkTarget = {
  id: string
  slug: string
  path: string
}

// 路径重构只声明重命名和移动两种动作。
export type PageRefactorKind = 'rename' | 'move'

// 受影响页面记录来源定位、匹配路径和警告，供重构确认界面展示。
export type PageRefactorAffectedPage = {
  fromPageId: string
  fromTitle: string
  fromPath: string
  matchedPaths: string[]
  warnings: string[]
}

// 预览返回路径变化与影响计数，不在本层直接执行链接替换。
export type PageRefactorPreview = {
  kind: PageRefactorKind
  pageId: string
  oldPath: string
  newPath: string
  affectedPages: PageRefactorAffectedPage[]
  counts: {
    affectedPages: number
    matchedLinks: number
  }
  warnings: string[]
}

// 获取树的根节点，保留服务端节点顺序和全部层级。
export async function fetchTree(): Promise<PageNode> {
  return await fetchWithAuth<PageNode>(`/api/tree`)
}

// 查询指定父节点下的路径建议，标题编码为查询值，父 ID 和当前 ID 直接拼接。
// currentId 为空时不发送该参数，不在本地推导可用名称。
export async function suggestSlug(
  parentId: string,
  title: string,
  currentId?: string,
): Promise<string> {
  if (!currentId) currentId = ''

  const data = await fetchWithAuth<{ slug: string }>(
    `/api/pages/slug-suggestion?parentId=${parentId}&title=${encodeURIComponent(title)}${currentId ? `&currentId=${currentId}` : ''}`,
  )
  return data.slug
}

// 页面路径编码为单个查询参数，可传 AbortSignal 取消请求，响应不作运行时结构校验。
export async function getPageByPath(
  path: string,
  signal?: AbortSignal,
): Promise<Page> {
  return await fetchWithAuth<Page>(
    `/api/pages/by-path?path=${encodeURIComponent(path)}`,
    { signal },
  )
}

// 稳定 ID 编码为一个路径片段，交由服务端解析当前页面定位。
export async function getPermalinkTarget(id: string): Promise<PermalinkTarget> {
  return await fetchWithAuth<PermalinkTarget>(
    `/api/pages/permalink/${encodeURIComponent(id)}`,
  )
}

// 空字符串父 ID 转为 null，其余字段按原值提交，不在此填充正文或推断页面类型。
export async function createPage({
  title,
  slug,
  parentId,
  kind,
}: {
  title: string
  slug: string
  parentId: string | null
  kind: PageKind
}) {
  if (parentId === '') parentId = null

  return await fetchWithAuth(`/api/pages`, {
    method: 'POST',
    body: JSON.stringify({ title, slug, parentId, kind }),
  })
}

// 复制请求将空字符串与 root 父 ID 统一转为 null，目标标题和 slug 映射到接口字段。
// 源 ID 直接放入路径，复制范围和数据继承由服务端决定。
export async function copyPage(
  id: string,
  targetParentId: string | null,
  targetTitle: string,
  targetSlug: string,
) {
  if (targetParentId === '' || targetParentId === 'root') targetParentId = null
  return await fetchWithAuth(`/api/pages/copy/${id}`, {
    method: 'POST',
    body: JSON.stringify({
      targetParentId,
      title: targetTitle,
      slug: targetSlug,
    }),
  })
}

// 同时提交当前版本与完整编辑字段，版本冲突交由服务端及统一错误处理返回。
// 返回值允许 null，本函数不把空响应替换为输入构造的页面。
export async function updatePage(
  id: string,
  version: string,
  title: string,
  slug: string,
  content: string,
  tags: string[],
  properties: Record<string, string>,
): Promise<Page | null> {
  return await fetchWithAuth<Page | null>(`/api/pages/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ version, title, slug, content, tags, properties }),
  })
}

// 删除请求显式发送 recursive 文本，运行时传入 undefined 时按 false 处理。
// version 非空才加入查询，不在客户端验证递归删除范围。
export async function deletePage(
  id: string,
  recursive: boolean,
  version: string,
) {
  if (recursive === undefined) recursive = false

  const params = new URLSearchParams({
    recursive: recursive ? 'true' : 'false',
  })
  if (version) params.set('version', version)

  return await fetchWithAuth(`/api/pages/${id}?${params.toString()}`, {
    method: 'DELETE',
  })
}

// 移动请求将空字符串或 root 父级转为 null，position 为 undefined 时完全省略该字段。
// 不在本层重排兄弟节点，目标位置合法性由服务端校验。
export async function movePage(
  id: string,
  version: string,
  parentId: string | null,
  position?: number,
) {
  if (parentId === '' || parentId == 'root') parentId = null

  return await fetchWithAuth(`/api/pages/${id}/move`, {
    method: 'PUT',
    body: JSON.stringify(
      position === undefined
        ? { version, parentId }
        : { version, parentId, position },
    ),
  })
}

// 预览输入用 kind 区分重命名与移动，原样 POST，不提交版本或执行链接改写。
export async function previewPageRefactor(
  id: string,
  payload:
    | {
        kind: 'rename'
        title: string
        slug: string
      }
    | {
        kind: 'move'
        parentId: string | null
      },
): Promise<PageRefactorPreview> {
  return await fetchWithAuth<PageRefactorPreview>(
    `/api/pages/${id}/refactor/preview`,
    {
      method: 'POST',
      body: JSON.stringify(payload),
    },
  )
}

// 应用重构同时提交版本和 rewriteLinks 决策，重命名还携带正文，移动携带目标父级。
// 与 movePage 不同，这里不自行归一父节点值，完全保留 payload 的 JSON 表示。
export async function applyPageRefactor(
  id: string,
  payload:
    | {
        kind: 'rename'
        version: string
        title: string
        slug: string
        content: string
        rewriteLinks: boolean
      }
    | {
        kind: 'move'
        version: string
        parentId: string | null
        rewriteLinks: boolean
        position?: number
      },
): Promise<Page | null> {
  return await fetchWithAuth<Page | null>(`/api/pages/${id}/refactor/apply`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

// 用版本和布尔值请求修改置顶状态，最终页面与排序由服务端返回和调用方刷新。
export async function pinPage(
  id: string,
  version: string,
  pinned: boolean,
): Promise<Page> {
  return await fetchWithAuth<Page>(`/api/pages/${id}/pin`, {
    method: 'PUT',
    body: JSON.stringify({ version, pinned }),
  })
}

// 空父 ID 在排序接口中使用 root 路径标识，orderedIDs 保持调用方顺序发送。
export async function sortPages(parentId: string, orderedIDs: string[]) {
  if (parentId === '') parentId = 'root'

  return await fetchWithAuth(`/api/pages/${parentId}/sort`, {
    method: 'PUT',
    body: JSON.stringify({ orderedIDs }),
  })
}

// 提交目标页面类型与当前版本，转换后内容和子项规则由服务端处理。
export async function convertPage(
  id: string,
  targetKind: PageKind,
  version: string,
) {
  return await fetchWithAuth(`/api/pages/convert/${id}`, {
    method: 'POST',
    body: JSON.stringify({ targetKind, version }),
  })
}

// 路径查询同时返回整体存在和可创建状态，以及各段路径已解析到的页面 ID。
export type PathLookupResult = {
  path: string
  exists: boolean
  canCreate: boolean
  segments: { slug: string; id?: string; exists: boolean }[]
}

// 将原路径编码到查询参数，不在发送前拆分或清理点目录。
export async function lookupPath(path: string): Promise<PathLookupResult> {
  return await fetchWithAuth<PathLookupResult>(
    `/api/pages/lookup?path=${encodeURIComponent(path)}`,
  )
}

// 请求确保指定路径页面存在，目标标题使用 title 字段，不自行创建任何中间节点。
export async function ensurePage(path: string, targetTitle: string) {
  return await fetchWithAuth(`/api/pages/ensure`, {
    method: 'POST',
    body: JSON.stringify({ path, title: targetTitle }),
  })
}
