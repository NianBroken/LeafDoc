// 页面变更确认只决定是否重写链接，实际页面提交和链接操作由调用方执行。
import { PageRefactorPreview } from '@/shared/lib/api/pages'
import { DIALOG_PAGE_REFACTOR_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useConfigStore } from '@/shared/state/config'
import { useDialogsStore } from '@/shared/state/dialogs'

type ConfirmPageRefactorOptions = {
  allowSkipRewrite?: boolean
}

// 关闭链接重构功能，或没有受影响页面且没有警告时，直接返回 false 而不开弹窗。
// 弹窗回调的 true、false 和 null 原样返回，分别留给调用方解释处理。
export function confirmPageRefactor(
  preview: PageRefactorPreview,
  options?: ConfirmPageRefactorOptions,
): Promise<boolean | null> {
  if (!useConfigStore.getState().enableLinkRefactor) {
    return Promise.resolve(false)
  }

  const affectedPages = preview?.counts?.affectedPages ?? 0
  const warnings = preview?.warnings ?? []

  if (affectedPages === 0 && warnings.length === 0) {
    return Promise.resolve(false)
  }

  // 只在需要确认时创建 Promise，允许跳过重写的选项默认 false，没有自行超时逻辑。
  return new Promise((resolve) => {
    useDialogsStore.getState().openDialog(DIALOG_PAGE_REFACTOR_CONFIRMATION, {
      preview,
      allowSkipRewrite: options?.allowSkipRewrite ?? false,
      onResolve: (rewriteLinks: boolean | null) => {
        resolve(rewriteLinks)
      },
    })
  })
}
