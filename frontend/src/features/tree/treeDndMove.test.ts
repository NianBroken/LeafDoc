// 通过接口替身验证跨父节点移动的普通路径、确认流程、取消结果和根节点转换。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  applyPageRefactor,
  movePage,
  previewPageRefactor,
} from '@/shared/lib/api/pages'
import type { PageRefactorPreview } from '@/shared/lib/api/pages'
import { confirmPageRefactor } from '@/features/page/pageRefactorDialogState'
import { useConfigStore } from '@/shared/state/config'
import { performTreeCrossParentMove } from '@/features/tree/treeDndMove'
import { ROOT_ID } from '@/features/tree/treeDndUtils'

vi.mock('@/shared/lib/api/pages', async () => {
  const actual = await vi.importActual<typeof import('@/shared/lib/api/pages')>(
    '@/shared/lib/api/pages',
  )
  return {
    ...actual,
    movePage: vi.fn(),
    previewPageRefactor: vi.fn(),
    applyPageRefactor: vi.fn(),
  }
})

vi.mock('@/features/page/pageRefactorDialogState', () => ({
  confirmPageRefactor: vi.fn(),
}))

const dragged = { id: 'page-1', version: 'v1' }
// 预览数据包含一个受影响页面，另一个副本模拟没有引用需要重写的结果。
const preview: PageRefactorPreview = {
  kind: 'move',
  pageId: 'page-1',
  oldPath: 'section-a/page-1',
  newPath: 'section-b/page-1',
  affectedPages: [
    {
      fromPageId: 'linker-1',
      fromTitle: 'Linker',
      fromPath: 'linker',
      matchedPaths: ['/section-a/page-1'],
      warnings: [],
    },
  ],
  counts: { affectedPages: 1, matchedLinks: 1 },
  warnings: [],
}
const noAffectedPreview: PageRefactorPreview = {
  ...preview,
  affectedPages: [],
  counts: { affectedPages: 0, matchedLinks: 0 },
}

describe('performTreeCrossParentMove', () => {
  beforeEach(() => {
    // 清空接口调用记录，配置开关由每个用例明确设置。
    vi.clearAllMocks()
  })

  it('calls plain movePage when link refactor is disabled', async () => {
    // 关闭开关时直接提交目标父节点与索引，并确认不请求预览。
    useConfigStore.setState({ enableLinkRefactor: false })
    vi.mocked(movePage).mockResolvedValue(null)

    const result = await performTreeCrossParentMove(dragged, {
      parentId: 'section-b',
      index: 2,
    })

    expect(movePage).toHaveBeenCalledWith('page-1', 'v1', 'section-b', 2)
    expect(previewPageRefactor).not.toHaveBeenCalled()
    expect(result).toEqual({ status: 'moved', preview: null })
  })

  it('applies with rewriteLinks:false when nothing is affected (no dialog shown)', async () => {
    // 无受影响链接时确认流程返回 false，执行请求仍携带该值与目标位置。
    useConfigStore.setState({ enableLinkRefactor: true })
    vi.mocked(previewPageRefactor).mockResolvedValue(noAffectedPreview)
    vi.mocked(confirmPageRefactor).mockResolvedValue(false)
    vi.mocked(applyPageRefactor).mockResolvedValue(null)

    const result = await performTreeCrossParentMove(dragged, {
      parentId: 'section-b',
      index: 2,
    })

    expect(previewPageRefactor).toHaveBeenCalledWith('page-1', {
      kind: 'move',
      parentId: 'section-b',
    })
    expect(confirmPageRefactor).toHaveBeenCalledWith(noAffectedPreview, {
      allowSkipRewrite: true,
    })
    expect(applyPageRefactor).toHaveBeenCalledWith('page-1', {
      kind: 'move',
      version: 'v1',
      parentId: 'section-b',
      position: 2,
      rewriteLinks: false,
    })
    expect(movePage).not.toHaveBeenCalled()
    expect(result).toEqual({ status: 'moved', preview: noAffectedPreview })
  })

  it('applies with rewriteLinks:true when the user confirms the rewrite', async () => {
    // 用户确认后执行接口明确接收 true，结果保留预览以供调用者更新界面。
    useConfigStore.setState({ enableLinkRefactor: true })
    vi.mocked(previewPageRefactor).mockResolvedValue(preview)
    vi.mocked(confirmPageRefactor).mockResolvedValue(true)
    vi.mocked(applyPageRefactor).mockResolvedValue(null)

    const result = await performTreeCrossParentMove(dragged, {
      parentId: 'section-b',
      index: 2,
    })

    expect(applyPageRefactor).toHaveBeenCalledWith(
      'page-1',
      expect.objectContaining({ rewriteLinks: true }),
    )
    expect(result).toEqual({ status: 'moved', preview })
  })

  it('aborts without calling apply or movePage when the user cancels', async () => {
    // 空确认结果代表取消，预览之后不调用任一移动提交接口。
    useConfigStore.setState({ enableLinkRefactor: true })
    vi.mocked(previewPageRefactor).mockResolvedValue(preview)
    vi.mocked(confirmPageRefactor).mockResolvedValue(null)

    const result = await performTreeCrossParentMove(dragged, {
      parentId: 'section-b',
      index: 2,
    })

    expect(applyPageRefactor).not.toHaveBeenCalled()
    expect(movePage).not.toHaveBeenCalled()
    expect(result).toEqual({ status: 'aborted' })
  })

  it('normalizes the root sentinel to null for the refactor endpoints', async () => {
    // 根节点占位符只在预览与执行参数中转换为 null。
    useConfigStore.setState({ enableLinkRefactor: true })
    vi.mocked(previewPageRefactor).mockResolvedValue(preview)
    vi.mocked(confirmPageRefactor).mockResolvedValue(false)
    vi.mocked(applyPageRefactor).mockResolvedValue(null)

    await performTreeCrossParentMove(dragged, { parentId: ROOT_ID, index: 0 })

    expect(previewPageRefactor).toHaveBeenCalledWith('page-1', {
      kind: 'move',
      parentId: null,
    })
    expect(applyPageRefactor).toHaveBeenCalledWith(
      'page-1',
      expect.objectContaining({ parentId: null }),
    )
  })
})
