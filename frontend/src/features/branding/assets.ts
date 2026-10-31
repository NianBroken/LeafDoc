// 为品牌预览和页面图标统一拼接资源地址，站点挂载前缀由路由工具处理。
import { withBasePath } from '@/shared/lib/routePath'

// 未指定自定义文件时使用随应用提供的标志与图标，文件名按资源种类选择。
const BUILTIN_FILES = {
  logo: 'leafdoc-logo.webp',
  favicon: 'leafdoc-favicon.ico',
}

export function brandingAssetUrl(
  kind: 'logo' | 'favicon',
  file: string,
  version: number,
): string {
  // 非空文件名指向品牌资源接口，空文件名指向内置文件；此处不检查资源是否存在。
  const path = withBasePath(
    file ? `/branding/${file}` : `/defaults/${BUILTIN_FILES[kind]}`,
  )
  // 版本作为查询参数参与浏览器缓存键，调用方更换版本即可请求不同地址。
  return `${path}?v=${version}`
}
