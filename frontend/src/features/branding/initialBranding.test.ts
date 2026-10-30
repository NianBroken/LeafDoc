// Initial metadata preserves the real site name before API requests complete.
import { afterEach, describe, expect, it } from 'vitest'
import { readInitialBranding } from './initialBranding'

afterEach(() => {
  document.querySelector('meta[name="leafdoc-branding"]')?.remove()
  document.title = ''
})
describe('initial branding', () => {
  it('uses the current server name and resource files before loading APIs', () => {
    const meta = document.createElement('meta')
    meta.name = 'leafdoc-branding'
    meta.content = JSON.stringify({
      siteName: '文档 <站点> "名称"',
      logoFile: 'logo.png',
      faviconFile: 'icon.png',
    })
    document.head.append(meta)
    expect(readInitialBranding()).toEqual({
      siteName: '文档 <站点> "名称"',
      logoFile: 'logo.png',
      faviconFile: 'icon.png',
    })
  })
  it('preserves a valid document title when metadata is unavailable', () => {
    document.title = '实际站点'
    expect(readInitialBranding().siteName).toBe('实际站点')
  })
  it('never treats template tokens as a site name', () => {
    document.title = '{{__SITE_NAME__}}'
    expect(readInitialBranding().siteName).toBe('LeafDoc')
  })
})
