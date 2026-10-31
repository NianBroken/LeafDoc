// 本文件统一处理页面附件地址和缓存版本参数，不负责实际下载资源。
import { withBasePath } from '@/shared/lib/routePath'

// 版本处理只识别这三种原始前缀，内置 defaults 资源不在版本判断范围内。
function isAssetSrc(src: string): boolean {
  return (
    src.startsWith('/assets/') ||
    src.startsWith('assets/') ||
    src.startsWith('/api/')
  )
}

/**
 * 为根路径附件、接口和内置资源加上部署基础路径，相对 assets/ 先补前导斜线。
 * 其他文本原样返回，不解析外部 URL、查询参数或 data URI，也不检查地址是否已经带基础路径。
 */
export function normalizeAssetSrc(src: string): string {
  if (
    src.startsWith('/assets/') ||
    src.startsWith('/api/') ||
    src.startsWith('/defaults/')
  ) {
    return withBasePath(src)
  }

  if (src.startsWith('assets/')) {
    return withBasePath(`/${src}`)
  }

  return src
}

/**
 * 先规范资源路径，再对识别为附件或接口的原始地址添加当前毫秒时间戳版本。
 * 已有 v 参数时保留原值，URL 处理保留其他查询参数和片段，结果返回绝对地址。
 * 内置资源及其他来源只执行规范化，解析失败同样返回规范化地址，不抛出解析错误。
 */
export function versionAssetSrc(src: string): string {
  const normalized = normalizeAssetSrc(src)

  if (!isAssetSrc(src)) {
    return normalized
  }

  try {
    const url = new URL(normalized, location.origin)
    if (!url.searchParams.has('v')) {
      url.searchParams.set('v', Date.now().toString())
    }
    return url.toString()
  } catch {
    return normalized
  }
}
