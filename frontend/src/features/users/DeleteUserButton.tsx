import { Button } from '@/shared/components/ui/button'
import { User } from '@/shared/lib/api/users'
import { DIALOG_DELETE_USER_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useSessionStore } from '@/shared/state/session'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem } from '@/shared/components/ui/dropdown-menu'

type DeleteUserButtonProps = {
  user: User
  menuItem?: boolean
}

export function DeleteUserButton({ user, menuItem }: DeleteUserButtonProps) {
  const { t } = useTranslation('users')
  const openDialog = useDialogsStore((s) => s.openDialog)

  const { user: currentUser } = useSessionStore()
  if (currentUser?.id === user.id) return null

  const handleClick = () =>
    openDialog(DIALOG_DELETE_USER_CONFIRMATION, {
      userId: user.id,
      username: user.username,
    })

  if (menuItem) {
    return (
      <DropdownMenuItem className="text-error" onSelect={handleClick}>
        {t('actions.delete')}
      </DropdownMenuItem>
    )
  }

  return (
    <Button size="sm" variant="destructive" onClick={handleClick}>
      {t('actions.delete')}
    </Button>
  )
}
