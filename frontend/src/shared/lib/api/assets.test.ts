// 本文件验证附件路径整理和页面列表缓存，网络层只使用受控响应替身。
// 缓存测试检查数组引用、并发请求数、失败后重试及上传后的重新读取。
import { beforeEach, describe, expect, it, vi } from 'vitest'

// 统一拦截附件接口，调用历史用于区分缓存命中与真正发起请求。
const mockFetchWithAuth = vi.fn()
vi.mock('@/shared/lib/api/client', () => ({
  fetchWithAuth: (...args: unknown[]) => mockFetchWithAuth(...args),
}))

import {
  getPageAttachments,
  invalidatePageAttachments,
  listPageAttachments,
  uploadAsset,
} from '@/shared/lib/api/assets'

describe('listPageAttachments', () => {
  // 带根斜线与不带根斜线的 assets 路径均转换为站内地址，非图片附件也保留。
  it('keeps all assets and normalizes URLs', () => {
    expect(
      listPageAttachments(['/assets/p1/notes.pdf', 'assets/p1/guide.docx']),
    ).toEqual([
      { name: 'notes.pdf', url: '/assets/p1/notes.pdf' },
      { name: 'guide.docx', url: '/assets/p1/guide.docx' },
    ])
  })

  // 路径规范后相同的 PDF 合并为一项，图片继续保留且顺序不变。
  it('keeps images and deduplicates URLs', () => {
    expect(
      listPageAttachments([
        '/assets/p1/photo.png',
        '/assets/p1/notes.pdf',
        'assets/p1/notes.pdf',
      ]),
    ).toEqual([
      { name: 'photo.png', url: '/assets/p1/photo.png' },
      { name: 'notes.pdf', url: '/assets/p1/notes.pdf' },
    ])
  })
})

describe('getPageAttachments — caching', () => {
  beforeEach(() => {
    // 清理请求历史与本页缓存，不重新加载模块，以便测试同一缓存实例。
    mockFetchWithAuth.mockReset()
    invalidatePageAttachments('p1')
  })

  // 连续读取应复用同一个数组对象，而不只是生成内容相等的数组。
  it('fetches once and serves later reads from the cache', async () => {
    mockFetchWithAuth.mockResolvedValue({ files: ['/assets/p1/notes.pdf'] })

    const first = await getPageAttachments('p1')
    const second = await getPageAttachments('p1')

    expect(first).toEqual([{ name: 'notes.pdf', url: '/assets/p1/notes.pdf' }])
    expect(second).toBe(first)
    expect(mockFetchWithAuth).toHaveBeenCalledTimes(1)
  })

  // 两个读取在前一个请求完成前同时发起，请求数必须保持为一。
  it('deduplicates concurrent reads into a single request', async () => {
    mockFetchWithAuth.mockResolvedValue({ files: ['/assets/p1/notes.pdf'] })

    const [a, b] = await Promise.all([
      getPageAttachments('p1'),
      getPageAttachments('p1'),
    ])

    expect(a).toEqual(b)
    expect(mockFetchWithAuth).toHaveBeenCalledTimes(1)
  })

  // 首次拒绝不能成为持久缓存，后续调用重新请求并获得正常附件列表。
  it('refetches after a failed request instead of caching the failure', async () => {
    mockFetchWithAuth.mockRejectedValueOnce(new Error('boom'))
    await expect(getPageAttachments('p1')).rejects.toThrow('boom')

    mockFetchWithAuth.mockResolvedValue({ files: ['/assets/p1/notes.pdf'] })
    const files = await getPageAttachments('p1')

    expect(files).toEqual([{ name: 'notes.pdf', url: '/assets/p1/notes.pdf' }])
    expect(mockFetchWithAuth).toHaveBeenCalledTimes(2)
  })

  // 先缓存空列表，再完成上传，下一次列表调用必须读取服务端提供的新附件。
  it('busts the cache when an asset is uploaded for the page', async () => {
    mockFetchWithAuth.mockResolvedValueOnce({ files: [] })
    expect(await getPageAttachments('p1')).toEqual([])

    mockFetchWithAuth.mockResolvedValueOnce({ file: '/assets/p1/notes.pdf' })
    await uploadAsset('p1', new File(['x'], 'notes.pdf'))

    mockFetchWithAuth.mockResolvedValueOnce({ files: ['/assets/p1/notes.pdf'] })
    expect(await getPageAttachments('p1')).toEqual([
      { name: 'notes.pdf', url: '/assets/p1/notes.pdf' },
    ])
  })
})
