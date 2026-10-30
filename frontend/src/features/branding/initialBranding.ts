// The server-provided metadata supplies branding before the first render.
import { DEFAULT_SITE_NAME } from '@/shared/lib/brandDefaults'

export type InitialBranding = {
  siteName: string
  logoFile: string
  faviconFile: string
}
export function readInitialBranding(): InitialBranding {
  const raw = document
    .querySelector('meta[name="leafdoc-branding"]')
    ?.getAttribute('content')
  if (raw) {
    try {
      const value: unknown = JSON.parse(raw)
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
      // Invalid metadata cannot supply executable content or replace the actual document title.
    }
  }
  const title = document.title.trim()
  return {
    siteName: title && !title.includes('{{') ? title : DEFAULT_SITE_NAME,
    logoFile: '',
    faviconFile: '',
  }
}
