import { buildViewUrl } from '@/shared/lib/routePath'

/**
 * 本文件处理页面路径、父路径、树查询键和相对链接，不直接读取部署前缀配置。
 * 路由类型转换交给 routePath.ts，再对结果执行页面路径规范化。
 */

/**
 * 按文本去掉查询串和片段，保证前导斜线并去掉非根路径的末尾斜线。
 * 不合并路径中间的连续斜线，不解析点目录，也不进行 URL 解码。
 */
export function normalizeWikiRoutePath(path: string): string {
  let normalized = path.split('?')[0].split('#')[0]

  if (!normalized.startsWith('/')) {
    normalized = `/${normalized}`
  }

  if (normalized.length > 1) {
    normalized = normalized.replace(/\/+$/, '')
  }

  return normalized
}

/**
 * 先规范页面路径，再移除唯一的前导斜线，根路径对应空查询键。
 * 输入中的额外前导斜线不在规范函数中合并，因此此处只删除第一个字符。
 */
export function toWikiLookupPath(path: string): string {
  const normalized = normalizeWikiRoutePath(path)
  return normalized === '/' ? '' : normalized.slice(1)
}

/**
 * 将支持的编辑、历史和查看路由统一转为带前导斜线的页面路径。
 * buildViewUrl 负责路由前缀，当前规范步骤继续去掉查询串、片段和末尾斜线。
 */
export function getWikiTargetRoutePath(pathname: string): string {
  return normalizeWikiRoutePath(buildViewUrl(pathname))
}

/**
 * 把当前页面路径当作目录基准，使用 URL 规则解析相对目标，再只取 pathname。
 * 外部目标也仅保留路径，协议、主机、查询串和片段不在结果中，非法 URL 会直接抛错。
 */
export function resolveWikiLinkPath(currentPath: string, href: string): string {
  const normalizedCurrentPath = normalizeWikiRoutePath(currentPath)
  const folderBase = normalizedCurrentPath.endsWith('/')
    ? normalizedCurrentPath
    : `${normalizedCurrentPath}/`

  const base = new URL(folderBase, 'https://leafdoc.local')
  const url = new URL(href, base)

  return normalizeWikiRoutePath(url.pathname)
}

/**
 * 去掉查询串和片段后按非空斜线片段取父路径，根与顶层页面均返回 /。
 * 分段时过滤空串，因此返回父路径不会保留输入中的中间重复斜线。
 */
export function getParentWikiRoutePath(path: string): string {
  const normalized = normalizeWikiRoutePath(path)

  if (normalized === '/') {
    return '/'
  }

  const segments = normalized.split('/').filter(Boolean)
  if (segments.length <= 1) {
    return '/'
  }

  return `/${segments.slice(0, -1).join('/')}`
}

/**
 * 当前查看页面等于删除目标或位于其下级时，返回删除目标的父路径。
 * 无需跳转时返回规范后的原路由，保留其编辑或历史前缀，但去掉查询串和片段。
 */
export function getDeleteRedirectRoutePath(
  currentLocationPath: string,
  deletedPagePath: string,
): string {
  const currentRoutePath = normalizeWikiRoutePath(currentLocationPath)
  const currentViewPath = normalizeWikiRoutePath(buildViewUrl(currentRoutePath))
  const deletedRoutePath = normalizeWikiRoutePath(deletedPagePath)

  const isDeletedRouteActive =
    currentViewPath === deletedRoutePath ||
    currentViewPath.startsWith(`${deletedRoutePath}/`)

  if (isDeletedRouteActive) {
    return getParentWikiRoutePath(deletedRoutePath)
  }

  return currentRoutePath
}
