import { Button } from '@/shared/components/ui/button'
import { User } from '@/shared/lib/api/users'
import { DIALOG_DELETE_USER_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useSessionStore } from '@/shared/state/session'
import { useTranslation } from 'react-i18next'

type DeleteUserButtonProps = {
  user: User
}

export function DeleteUserButton({ user }: DeleteUserButtonProps) {
  const { t } = useTranslation('users')
  const openDialog = useDialogsStore((s) => s.openDialog)

  const { user: currentUser } = useSessionStore()
  if (currentUser?.id === user.id) return null

  return (
    <Button
      size="sm"
      variant="destructive"
      onClick={() =>
        openDialog(DIALOG_DELETE_USER_CONFIRMATION, {
          userId: user.id,
          username: user.username,
        })
      }
    >
      {t('actions.delete')}
    </Button>
  )
}
