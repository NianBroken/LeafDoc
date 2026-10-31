// 本文件处理浏览器部署前缀和编辑、历史、永久链接路由，不解析 Markdown 相对链接。
import { BASE_PATH } from '@/shared/lib/config'

/**
 * 将空值和根路径归一为空前缀，其余输入补前导斜线并移除末尾连续斜线。
 * 不去掉两端空白，不合并中间重复斜线，也不对输入执行 URL 解码。
 */
export function normalizeBasePath(base: string | undefined | null): string {
  if (!base || base === '/') return ''
  base = base.startsWith('/') ? base : `/${base}`
  return base.replace(/\/+$/, '')
}

// 模块加载时固定基础路径，后续工具函数共用同一归一结果。
const BASE = normalizeBasePath(BASE_PATH)

/** 空路径转成根路径，非空且没有前导斜线时补一个，其余文本不变。 */
function ensureLeadingSlash(pathname: string): string {
  if (!pathname) {
    return '/'
  }
  return pathname.startsWith('/') ? pathname : `/${pathname}`
}

/** 加上部署前缀，不检查输入是否已经包含前缀；根部署时输入原样返回。 */
export function withBasePath(p: string): string {
  if (!BASE) return p
  return BASE + (p.startsWith('/') ? p : `/${p}`)
}

/**
 * 先保证路径有前导斜线，再按完整基础路径或带斜线的子路径边界去掉部署前缀。
 * 根部署接受所有路径，有配置前缀但不匹配时返回 null，不按相似文本截断。
 */
export function stripBasePath(pathname: string): string | null {
  pathname = ensureLeadingSlash(pathname)
  if (!BASE) return pathname
  if (pathname === BASE) return '/'
  if (pathname.startsWith(BASE + '/')) {
    const stripped = pathname.slice(BASE.length)
    return ensureLeadingSlash(stripped)
  }
  return null
}

/**
 * 生成编辑路由，能识别部署前缀时先去掉，已有 /e/ 则保留。
 * 输入不属于部署前缀时把其整体作为页面路径并加 /e，不丢弃输入中的目录。
 * 不编码页面片段，也不检查页面是否存在。
 */
export function buildEditUrl(pathname: string): string {
  let p = stripBasePath(pathname)
  if (p === null) return `/e${ensureLeadingSlash(pathname)}`
  if (p.startsWith('/e/')) return p
  if (p.startsWith('/')) {
    p = p.slice(1)
  }

  return `/e/${p}`
}

/**
 * 生成历史路由，已有 /history/ 子路径直接保留，历史根统一带末尾斜线。
 * 无法剥离部署前缀时直接给完整输入加历史前缀，不解析查询串或片段。
 */
export function buildHistoryUrl(pathname: string): string {
  let p = stripBasePath(pathname)
  if (p === null) return `/history${ensureLeadingSlash(pathname)}`
  if (p === '/history' || p === '/history/') return '/history/'
  if (p.startsWith('/history/')) return p
  if (p.startsWith('/')) {
    p = p.slice(1)
  }

  return `/history/${p}`
}

/**
 * 将页面 ID 编码为 /p/ 路由，非空 slug 去掉两端空白后作为单个编码片段追加。
 * slug 不参与此处页面查找，缺失或仅空白时只返回 ID 路由。
 */
export function buildPermalinkPath(id: string, slug?: string): string {
  const encodedID = encodeURIComponent(id)
  const normalizedSlug = slug?.trim()

  if (!normalizedSlug) {
    return `/p/${encodedID}`
  }

  return `/p/${encodedID}/${encodeURIComponent(normalizedSlug)}`
}

/**
 * 生成带部署前缀的浏览器编辑地址，已是 /e/ 形式时直接添加部署前缀。
 * 其他路径先交给 buildEditUrl 处理，不对最终地址进行存在性校验。
 */
export function buildBrowserEditUrl(pathname: string): string {
  const normalized = ensureLeadingSlash(pathname)
  if (normalized.startsWith('/e/')) {
    return withBasePath(normalized)
  }
  return withBasePath(buildEditUrl(pathname))
}

/**
 * 尝试去掉部署前缀，再将编辑或历史路由转换成查看页面使用的路径。
 * /e/ 分支返回去掉整个前缀后的文本，因此通常没有前导斜线；历史分支保留该斜线。
 * 历史根返回 /，其他路径原样保留，调用方仍需按自身路径约定继续规范化。
 */
export function buildViewUrl(pathname: string): string {
  const stripped = stripBasePath(pathname)
  pathname = stripped ?? ensureLeadingSlash(pathname)

  if (pathname.startsWith('/e/')) return pathname.slice(3)
  if (pathname === '/history' || pathname === '/history/') return '/'
  if (pathname.startsWith('/history/')) return pathname.slice('/history'.length)
  return pathname
}
