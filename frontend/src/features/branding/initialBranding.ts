// 从服务端写入的页面元数据取得初始品牌配置，供状态容器在接口返回前使用。
import { DEFAULT_SITE_NAME } from '@/shared/lib/brandDefaults'

export type InitialBranding = {
  // 空资源文件名表示使用内置图片，站点名称始终由元数据、页面标题或默认常量提供。
  siteName: string
  logoFile: string
  faviconFile: string
}
export function readInitialBranding(): InitialBranding {
  // 只读取指定 meta 的 content，不从页面中执行脚本或解释模板内容。
  const raw = document
    .querySelector('meta[name="leafdoc-branding"]')
    ?.getAttribute('content')
  if (raw) {
    try {
      const value: unknown = JSON.parse(raw)
      // 必须有去除空白后非空的名称才能接受整份元数据，返回时保留名称原文。
      if (
        value &&
        typeof value === 'object' &&
        'siteName' in value &&
        typeof value.siteName === 'string' &&
        value.siteName.trim()
      ) {
        return {
          siteName: value.siteName,
          logoFile:
            'logoFile' in value && typeof value.logoFile === 'string'
              ? value.logoFile
              : '',
          faviconFile:
            'faviconFile' in value && typeof value.faviconFile === 'string'
              ? value.faviconFile
              : '',
        }
      }
    } catch {
      // JSON 解析失败后继续读取页面标题，不将无效内容写入返回结果。
    }
  }
  const title = document.title.trim()
  // 页面标题含模板起始标记时不用作站点名称，缺少可用标题时使用统一默认名称。
  return {
    siteName: title && !title.includes('{{') ? title : DEFAULT_SITE_NAME,
    logoFile: '',
    faviconFile: '',
  }
}
