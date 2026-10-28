import * as brandingAPI from '@/shared/lib/api/branding'
import i18next from '@/shared/lib/i18n'
import { brandingAssetUrl } from '../assets'
import { create } from 'zustand'

const t = (key: string) => i18next.t(key, { ns: 'branding' })

type BrandingStore = {
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
  siteName: 'LeafWiki',
  logoFile: '',
  faviconFile: '',
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
      set({
        error:
          err instanceof Error ? err.message : t('storeErrors.loadFallback'),
        isLoading: false,
      })
    }
  },

  updateBranding: async (config) => {
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
        error:
          err instanceof Error ? err.message : t('storeErrors.updateFallback'),
        isLoading: false,
      })
      throw err
    }
  },

  uploadLogo: async (file) => {
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
        error:
          err instanceof Error
            ? err.message
            : t('storeErrors.uploadLogoFallback'),
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },

  uploadFavicon: async (file) => {
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
        error:
          err instanceof Error
            ? err.message
            : t('storeErrors.uploadFaviconFallback'),
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },

  deleteLogo: async () => {
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
        error:
          err instanceof Error
            ? err.message
            : t('storeErrors.deleteLogoFallback'),
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },

  deleteFavicon: async () => {
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
        error:
          err instanceof Error
            ? err.message
            : t('storeErrors.deleteFaviconFallback'),
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },
}))

function applyFavicon(faviconFile: string, version: number) {
  const link = getOrCreateFaviconLink()
  const source = brandingAssetUrl('favicon', faviconFile, version)!
  link.href = source
  // Verify the image independently because link error events vary by browser.
  if (faviconFile) {
    const image = new Image()
    image.onerror = () => {
      if (link.getAttribute('href') === source) {
        link.href = brandingAssetUrl('favicon', '', version)!
      }
      image.onload = null
      image.onerror = null
    }
    image.onload = () => {
      image.onload = null
      image.onerror = null
    }
    image.src = source
  }
}

function getOrCreateFaviconLink(): HTMLLinkElement {
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
