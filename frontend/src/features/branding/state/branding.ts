// 集中维护品牌配置、资源版本及请求状态，并在图标变化时同步文档中的 icon 链接。
import { mapApiError } from '@/shared/lib/api/errors'
import * as brandingAPI from '@/shared/lib/api/branding'
import i18next from '@/shared/lib/i18n'
import { brandingAssetUrl } from '../assets'
import { readInitialBranding } from '../initialBranding'
import { create } from 'zustand'

const t = (key: string) => i18next.t(key, { ns: 'branding' })

type BrandingStore = {
  // 资源版本参与地址查询参数，扩展名与大小限制来自配置接口，供设置页面展示。
  siteName: string
  logoFile: string
  faviconFile: string
  logoVersion: number
  faviconVersion: number
  logoExts: string[]
  maxLogoSize: number
  faviconExts: string[]
  maxFaviconSize: number
  isLoaded: boolean
  isLoading: boolean
  error: string | null

  loadBranding: () => Promise<void>
  updateBranding: (config: Partial<brandingAPI.BrandingConfig>) => Promise<void>
  uploadLogo: (file: File) => Promise<void>
  uploadFavicon: (file: File) => Promise<void>
  deleteLogo: () => Promise<void>
  deleteFavicon: () => Promise<void>
}

export const useBrandingStore = create<BrandingStore>((set) => ({
  // 接口加载前使用 HTML 元数据初始化名称和文件；此时上传限制尚未取得。
  ...readInitialBranding(),
  logoVersion: 0,
  faviconVersion: 0,
  logoExts: [],
  maxLogoSize: 0,
  faviconExts: [],
  maxFaviconSize: 0,
  isLoaded: false,
  isLoading: false,
  error: null,

  loadBranding: async () => {
    // 成功后统一写入配置和限制，并使用同一时间戳标记两个资源地址。
    set({ isLoading: true, error: null })
    try {
      const config = await brandingAPI.getBranding()
      const assetVersion = Date.now()
      applyFavicon(config.faviconFile, assetVersion)
      set({
        siteName: config.siteName,
        logoFile: config.logoFile,
        logoVersion: assetVersion,
        logoExts: config.brandingConstraints.logoExts,
        maxLogoSize: config.brandingConstraints.maxLogoSize,
        faviconFile: config.faviconFile,
        faviconVersion: assetVersion,
        faviconExts: config.brandingConstraints.faviconExts,
        maxFaviconSize: config.brandingConstraints.maxFaviconSize,
        isLoaded: true,
        isLoading: false,
      })
    } catch (err) {
      // 加载错误仅保存到状态，不向调用者重新抛出，已有品牌字段保持原值。
      set({
        error: mapApiError(err, t('storeErrors.loadFallback')).message,
        isLoading: false,
      })
    }
  },

  updateBranding: async (config) => {
    // 接受部分配置，成功时以接口返回的品牌字段替换本地值，并刷新两个资源版本。
    set({ isLoading: true, error: null })
    try {
      const updated = await brandingAPI.updateBranding(config)
      const assetVersion = Date.now()
      applyFavicon(updated.faviconFile, assetVersion)
      set({
        logoVersion: assetVersion,
        faviconVersion: assetVersion,
        siteName: updated.siteName,
        logoFile: updated.logoFile,
        faviconFile: updated.faviconFile,
        isLoading: false,
      })
    } catch (err) {
      set({
        error: mapApiError(err, t('storeErrors.updateFallback')).message,
        isLoading: false,
      })
      throw err
    }
  },

  uploadLogo: async (file) => {
    // 标志上传只更新标志文件和版本，保留站点名称与图标状态。
    set({ isLoading: true, error: null })
    try {
      const result = await brandingAPI.uploadBrandingLogo(file)
      const assetVersion = Date.now()
      set({
        logoFile: result.branding.logoFile,
        logoVersion: assetVersion,
      })
    } catch (err) {
      set({
        error: mapApiError(err, t('storeErrors.uploadLogoFallback')).message,
      })
      throw err
    } finally {
      // 所有上传结果均结束加载状态，失败在记录可读消息后继续抛给界面处理。
      set({ isLoading: false })
    }
  },

  uploadFavicon: async (file) => {
    // 图标上传成功后同时更新浏览器图标链接和设置页预览所用状态。
    set({ isLoading: true, error: null })
    try {
      const result = await brandingAPI.uploadBrandingFavicon(file)
      const assetVersion = Date.now()
      applyFavicon(result.branding.faviconFile, assetVersion)
      set({
        faviconFile: result.branding.faviconFile,
        faviconVersion: assetVersion,
      })
    } catch (err) {
      set({
        error: mapApiError(err, t('storeErrors.uploadFaviconFallback')).message,
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },

  deleteLogo: async () => {
    // 使用删除响应中的文件字段，空文件名指向内置资源；版本仍参与缓存区分。
    set({ isLoading: true, error: null })
    try {
      const result = await brandingAPI.deleteBrandingLogo()
      const assetVersion = Date.now()
      set({
        logoFile: result.logoFile,
        logoVersion: assetVersion,
      })
    } catch (err) {
      set({
        error: mapApiError(err, t('storeErrors.deleteLogoFallback')).message,
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },

  deleteFavicon: async () => {
    // 删除响应决定当前图标文件，并立即同步文档链接，加载状态在 finally 中释放。
    set({ isLoading: true, error: null })
    try {
      const result = await brandingAPI.deleteBrandingFavicon()
      const assetVersion = Date.now()
      applyFavicon(result.faviconFile, assetVersion)
      set({
        faviconFile: result.faviconFile,
        faviconVersion: assetVersion,
      })
    } catch (err) {
      set({
        error: mapApiError(err, t('storeErrors.deleteFaviconFallback')).message,
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },
}))

function applyFavicon(faviconFile: string, version: number) {
  // 先更新文档链接，再独立尝试加载自定义图片，失败时回退到同版本内置图标。
  const link = getOrCreateFaviconLink()
  const source = brandingAssetUrl('favicon', faviconFile, version)!
  link.href = source
  // 图片对象的事件负责判断加载结果，不依赖 link 元素派发错误事件。
  if (faviconFile) {
    const image = new Image()
    image.onerror = () => {
      // 只有链接仍指向本次探测地址时才回退，避免迟到的错误覆盖后续资源选择。
      if (link.getAttribute('href') === source) {
        link.href = brandingAssetUrl('favicon', '', version)!
      }
      image.onload = null
      image.onerror = null
    }
    image.onload = () => {
      // 探测结束后解除两个事件回调，不保留闭包中的链接与地址引用。
      image.onload = null
      image.onerror = null
    }
    image.src = source
  }
}

function getOrCreateFaviconLink(): HTMLLinkElement {
  // 复用文档中第一个 rel 包含 icon 的链接；没有匹配项时创建并挂到 head。
  const existing = document.querySelector(
    "link[rel*='icon']",
  ) as HTMLLinkElement | null

  if (existing) {
    return existing
  }

  const link = document.createElement('link')
  link.rel = 'icon'
  document.head.appendChild(link)
  return link
}
