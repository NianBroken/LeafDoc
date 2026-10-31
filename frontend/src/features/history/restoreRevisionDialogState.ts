// 本模块把恢复确认弹窗的回调封装为 Promise，只收集确认结果，不执行恢复请求。
import { type Revision } from '@/shared/lib/api/revisions'

import { DIALOG_RESTORE_REVISION_CONFIRMATION } from '@/shared/lib/registries/instances'

import { useDialogsStore } from '@/shared/state/dialogs'

// 弹窗收到版本与当前路径，onResolve 的布尔值或 null 原样完成 Promise。
// Promise 没有超时或主动拒绝分支，弹窗关闭流程须负责调用传入的回调。
export function confirmRestoreRevision(
  revision: Revision,
  currentSlug: string,
): Promise<boolean | null> {
  return new Promise((resolve) => {
    useDialogsStore
      .getState()
      .openDialog(DIALOG_RESTORE_REVISION_CONFIRMATION, {
        revision,
        currentSlug,
        onResolve: (confirmed: boolean | null) => {
          resolve(confirmed)
        },
      })
  })
}
