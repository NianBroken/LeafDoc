// 本文件封装品牌配置及图片操作，不在本地更新外观状态或控制浏览器缓存版本。
import { fetchWithAuth } from '@/shared/lib/api/client'

// 前端读取的配置形状包含资源文件名和上传限制，不等同于完整服务端内部结构。
export type BrandingConfig = {
  siteName: string
  logoFile: string
  faviconFile: string
  brandingConstraints: {
    logoExts: string[]
    maxLogoSize: number
    faviconExts: string[]
    maxFaviconSize: number
  }
}

// 读取当前外观响应，实际可用资源选择由服务端处理。
export async function getBranding(): Promise<BrandingConfig> {
  return await fetchWithAuth<BrandingConfig>('/api/branding')
}

// 发送传入的局部对象，不在本层筛选哪些字段允许更新，服务端负责验证。
export async function updateBranding(
  config: Partial<BrandingConfig>,
): Promise<BrandingConfig> {
  return await fetchWithAuth<BrandingConfig>('/api/branding', {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

// 用 file 表单字段上传标志，返回服务端给出的路径与当前配置，不自行解码图片。
export async function uploadBrandingLogo(
  file: File,
): Promise<{ path: string; branding: BrandingConfig }> {
  const formData = new FormData()
  formData.append('file', file)

  return await fetchWithAuth<{ path: string; branding: BrandingConfig }>(
    '/api/branding/logo',
    {
      method: 'POST',
      body: formData,
      headers: {}, // 浏览器为表单生成包含分隔标记的内容类型。
    },
  )
}

// 上传站点图标使用独立接口，其他传输与标志上传共用统一请求规则。
export async function uploadBrandingFavicon(
  file: File,
): Promise<{ path: string; branding: BrandingConfig }> {
  const formData = new FormData()
  formData.append('file', file)

  return await fetchWithAuth<{ path: string; branding: BrandingConfig }>(
    '/api/branding/favicon',
    {
      method: 'POST',
      body: formData,
      headers: {}, // 不手写 multipart 内容类型，避免遗漏浏览器生成的分隔标记。
    },
  )
}

// 删除自定义标志并返回服务端配置，前端状态刷新由调用方执行。
export async function deleteBrandingLogo(): Promise<BrandingConfig> {
  return await fetchWithAuth<BrandingConfig>('/api/branding/logo', {
    method: 'DELETE',
  })
}

// 删除自定义站点图标，不在本层替换页面中的 favicon 标签。
export async function deleteBrandingFavicon(): Promise<BrandingConfig> {
  return await fetchWithAuth<BrandingConfig>('/api/branding/favicon', {
    method: 'DELETE',
  })
}
