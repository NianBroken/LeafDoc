// 检查初始化读取规则，分别覆盖有效元数据、页面标题和未展开的模板占位内容。
import { afterEach, describe, expect, it } from 'vitest'
import { readInitialBranding } from './initialBranding'

afterEach(() => {
  // 清除用例写入的元数据和标题，避免前一个来源遮挡后续回退分支。
  document.querySelector('meta[name="leafdoc-branding"]')?.remove()
  document.title = ''
})
describe('initial branding', () => {
  it('uses the current server name and resource files before loading APIs', () => {
    // 名称中的中文、尖括号和引号按 JSON 字符串读取，返回值保留原始内容。
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
