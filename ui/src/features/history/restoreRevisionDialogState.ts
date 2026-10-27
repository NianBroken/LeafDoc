import { type Revision } from '@/shared/lib/api/revisions'

import { DIALOG_RESTORE_REVISION_CONFIRMATION } from '@/shared/lib/registries/instances'

import { useDialogsStore } from '@/shared/state/dialogs'

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
