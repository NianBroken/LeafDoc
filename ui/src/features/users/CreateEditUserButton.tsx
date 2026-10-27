import { Button } from '@/shared/components/ui/button'
import { User } from '@/shared/lib/api/users'
import { DIALOG_USER_FORM } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTranslation } from 'react-i18next'
type CreateEditUserButtonProps = {
  user?: User
}

export function CreateEditUserButton({ user }: CreateEditUserButtonProps) {
  const { t } = useTranslation('users')
  const openDialog = useDialogsStore((s) => s.openDialog)
  return user ? (
    <Button
      size="sm"
      variant="outline"
      onClick={() => openDialog(DIALOG_USER_FORM, { user })}
    >
      {t('actions.edit')}
    </Button>
  ) : (
    <Button variant="default" onClick={() => openDialog(DIALOG_USER_FORM)}>
      {t('actions.new')}
    </Button>
  )
}
