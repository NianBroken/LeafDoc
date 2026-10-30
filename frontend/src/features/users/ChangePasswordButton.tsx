import { Button } from '@/shared/components/ui/button'
import { User } from '@/shared/lib/api/users'
import { DIALOG_CHANGE_USER_PASSWORD } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useSessionStore } from '@/shared/state/session'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem } from '@/shared/components/ui/dropdown-menu'

type ChangePasswordButtonProps = {
  user: User
  menuItem?: boolean
}

export function ChangePasswordButton({
  user,
  menuItem,
}: ChangePasswordButtonProps) {
  const { t } = useTranslation('users')
  const openDialog = useDialogsStore((s) => s.openDialog)

  // Own password is changed in the account settings, which asks for the
  // current one; this admin action is for other users only.
  const { user: currentUser } = useSessionStore()
  if (currentUser?.id === user.id) return null

  const handleClick = () =>
    openDialog(DIALOG_CHANGE_USER_PASSWORD, {
      userId: user.id,
      username: user.username,
    })

  if (menuItem) {
    return (
      <DropdownMenuItem onSelect={handleClick}>
        {t('actions.changePassword')}
      </DropdownMenuItem>
    )
  }

  return (
    <Button size="sm" variant="secondary" onClick={handleClick}>
      {t('actions.changePassword')}
    </Button>
  )
}
