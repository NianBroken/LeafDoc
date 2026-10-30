import { Button } from '@/shared/components/ui/button'
import { DIALOG_API_KEY_FORM } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTranslation } from 'react-i18next'

export function CreateApiKeyButton() {
  const { t } = useTranslation('apikeys')
  const openDialog = useDialogsStore((s) => s.openDialog)
  return (
    <Button variant="default" onClick={() => openDialog(DIALOG_API_KEY_FORM)}>
      {t('create.button')}
    </Button>
  )
}
