import { TrashIcon } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import type { ApiKey } from '@/shared/lib/api/apikeys'
import { DIALOG_DELETE_API_KEY_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTranslation } from 'react-i18next'

type DeleteApiKeyButtonProps = {
  apiKey: ApiKey
}

export function DeleteApiKeyButton({ apiKey }: DeleteApiKeyButtonProps) {
  const { t } = useTranslation('apikeys')
  const openDialog = useDialogsStore((s) => s.openDialog)

  return (
    <Button
      size="icon"
      className="h-7 w-7"
      aria-label={t('delete.button')}
      title={t('delete.button')}
      variant="destructive"
      onClick={() =>
        openDialog(DIALOG_DELETE_API_KEY_CONFIRMATION, {
          apiKeyId: apiKey.id,
          apiKeyName: apiKey.name,
        })
      }
    >
      <TrashIcon className="h-4 w-4" />
    </Button>
  )
}
