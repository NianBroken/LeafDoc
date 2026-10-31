// 跨父节点移动按配置选择普通移动或预览、确认、执行链接重写的完整流程。
import {
  applyPageRefactor,
  movePage,
  PageRefactorPreview,
  previewPageRefactor,
} from '@/shared/lib/api/pages'
import { confirmPageRefactor } from '@/features/page/pageRefactorDialogState'
import { useConfigStore } from '@/shared/state/config'
import { DropResolution, ROOT_ID } from '@/features/tree/treeDndUtils'

export type TreeCrossParentMoveResult =
  | { status: 'moved'; preview: PageRefactorPreview | null }
  | { status: 'aborted' }

// 启用链接重写时，先取得影响预览并等待确认，再使用原版本号和插入位置提交移动。
// 确认返回空值表示取消；跳过重写返回 false，仍会执行移动并返回预览信息。
export async function performTreeCrossParentMove(
  dragged: { id: string; version: string },
  resolution: DropResolution,
): Promise<TreeCrossParentMoveResult> {
  // 功能关闭时直接使用普通移动接口，不执行预览与确认步骤。
  if (!useConfigStore.getState().enableLinkRefactor) {
    await movePage(
      dragged.id,
      dragged.version,
      resolution.parentId,
      resolution.index,
    )
    return { status: 'moved', preview: null }
  }

  const parentId = resolution.parentId === ROOT_ID ? null : resolution.parentId
  // 根节点占位 ID 转换为接口使用的空父节点值。

  const preview = await previewPageRefactor(dragged.id, {
    kind: 'move',
    parentId,
  })
  const rewriteLinks = await confirmPageRefactor(preview, {
    allowSkipRewrite: true,
  })
  if (rewriteLinks === null) {
    return { status: 'aborted' }
  }

  await applyPageRefactor(dragged.id, {
    kind: 'move',
    version: dragged.version,
    parentId,
    position: resolution.index,
    rewriteLinks,
  })

  return { status: 'moved', preview }
}
