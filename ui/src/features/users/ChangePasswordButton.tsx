import { Button } from '@/shared/components/ui/button'
import { User } from '@/shared/lib/api/users'
import { DIALOG_CHANGE_USER_PASSWORD } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useSessionStore } from '@/shared/state/session'
import { useTranslation } from 'react-i18next'

type ChangePasswordButtonProps = {
  user: User
}

export function ChangePasswordButton({ user }: ChangePasswordButtonProps) {
  const { t } = useTranslation('users')
  const openDialog = useDialogsStore((s) => s.openDialog)

  // Own password is changed in the account settings, which asks for the
  // current one; this admin action is for other users only.
  const { user: currentUser } = useSessionStore()
  if (currentUser?.id === user.id) return null

  return (
    <Button
      size="sm"
      variant="secondary"
      onClick={() =>
        openDialog(DIALOG_CHANGE_USER_PASSWORD, {
          userId: user.id,
          username: user.username,
        })
      }
    >
      {t('actions.changePassword')}
    </Button>
  )
}
