import { PageRefactorPreview } from '@/shared/lib/api/pages'
import { DIALOG_PAGE_REFACTOR_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useConfigStore } from '@/shared/state/config'
import { useDialogsStore } from '@/shared/state/dialogs'

type ConfirmPageRefactorOptions = {
  allowSkipRewrite?: boolean
}

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
