// 附件地址测试区分部署路径规范化与缓存版本处理，断言不触发实际资源下载。
// 测试环境采用根部署，因此 /assets/ 与 /api/ 路径保持原有前缀。
import { describe, expect, it } from 'vitest'
import { normalizeAssetSrc, versionAssetSrc } from '@/shared/lib/assetSrc'

describe('normalizeAssetSrc', () => {
  it('leaves an absolute external URL unchanged', () => {
    // 外部完整地址不属于站内附件路径，协议、主机和文件名原样保留。
    expect(normalizeAssetSrc('https://example.com/doc.pdf')).toBe(
      'https://example.com/doc.pdf',
    )
  })

  it('leaves a data URI unchanged', () => {
    // 内联图片内容无需部署前缀，规范化结果必须与输入文本相同。
    const dataUri = 'data:image/png;base64,abc123'
    expect(normalizeAssetSrc(dataUri)).toBe(dataUri)
  })

  it('normalizes a /assets/-prefixed path', () => {
    // 具有前导斜线的附件路径在根部署中不重复添加斜线。
    expect(normalizeAssetSrc('/assets/foo.png')).toBe('/assets/foo.png')
  })

  it('adds a leading slash to an assets/-prefixed path (no leading slash)', () => {
    // 相对 assets/ 前缀先转换为站内根路径，再由部署路径函数处理。
    expect(normalizeAssetSrc('assets/foo.png')).toBe('/assets/foo.png')
  })

  it('normalizes a /api/-prefixed path', () => {
    // 接口提供的文件地址使用相同的部署路径规则，不改写接口后缀。
    expect(normalizeAssetSrc('/api/assets/foo.pdf')).toBe('/api/assets/foo.pdf')
  })
})

describe('versionAssetSrc', () => {
  it('appends a ?v= cache-busting param to a wiki asset src', () => {
    // 不固定系统时间，只核对附件路径和数字版本参数的结构。
    const versioned = versionAssetSrc('/assets/foo.png')
    expect(versioned).toContain('/assets/foo.png')
    expect(versioned).toMatch(/\?v=\d+$/)
  })

  it('does not double-append ?v= when the src already has one', () => {
    // 显式版本由调用方控制，已有值不能被当前时间覆盖。
    const versioned = versionAssetSrc('/assets/foo.png?v=123')
    expect(versioned).toMatch(/\/assets\/foo\.png\?v=123$/)
  })

  it('preserves a #fragment after the cache-busting param (PDF #page=N)', () => {
    // PDF 页码片段位于查询串之后，添加版本时仍保留原来的目标页。
    const versioned = versionAssetSrc('/assets/manual.pdf#page=3')
    expect(versioned).toMatch(/\/assets\/manual\.pdf\?v=\d+#page=3$/)
  })

  it('normalizes a /api/-prefixed asset src and versions it too', () => {
    // 接口前缀也参加版本处理，版本参数位于文件路径之后。
    const versioned = versionAssetSrc('/api/assets/foo.pdf')
    expect(versioned).toMatch(/\/api\/assets\/foo\.pdf\?v=\d+$/)
  })

  it('leaves an external URL normalized but unversioned', () => {
    // 外部资源不参加站内缓存失效，结果不引入查询参数。
    expect(versionAssetSrc('https://example.com/doc.pdf')).toBe(
      'https://example.com/doc.pdf',
    )
  })
})
