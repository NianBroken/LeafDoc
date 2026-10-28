import { useTranslation } from 'react-i18next'
import { MoreHorizontal } from 'lucide-react'
import { Button } from '@/shared/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/shared/components/ui/dropdown-menu'
import type { User } from '@/shared/lib/api/users'
import { useSessionStore } from '@/shared/state/session'
import { CreateEditUserButton } from './CreateEditUserButton'
import { ChangePasswordButton } from './ChangePasswordButton'
import { DeleteUserButton } from './DeleteUserButton'
import { ResendInviteButton } from './ResendInviteButton'

// Secondary actions use a portal menu so fixed table cells never hide controls.
export function UserRowActions({ user }: { user: User }) {
  const { t } = useTranslation('users')
  const currentUser = useSessionStore((state) => state.user)
  return (
    <div className="users__actions">
      <CreateEditUserButton user={user} />
      {(user.mustSetPassword || user.id !== currentUser?.id) && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              size="sm"
              variant="outline"
              aria-label={t('actions.moreFor', { username: user.username })}
            >
              <MoreHorizontal className="h-4 w-4" />
              {t('actions.more')}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {user.mustSetPassword && (
              <>
                <DropdownMenuLabel>{t('invite.pendingPill')}</DropdownMenuLabel>
                <ResendInviteButton user={user} menuItem />
              </>
            )}
            <ChangePasswordButton user={user} menuItem />
            <DeleteUserButton user={user} menuItem />
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  )
}
