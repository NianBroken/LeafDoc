import { useTranslation } from 'react-i18next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/shared/components/ui/dropdown-menu'
import type { User } from '@/shared/lib/api/users'
import { CreateEditUserButton } from './CreateEditUserButton'
import { ChangePasswordButton } from './ChangePasswordButton'
import { DeleteUserButton } from './DeleteUserButton'
import { ResendInviteButton } from './ResendInviteButton'

// One portaled action menu keeps every action reachable in narrow equal-width cells.
export function UserRowActions({ user }: { user: User }) {
  const { t } = useTranslation('users')
  return (
    <div className="users__actions">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="settings__text-action"
            aria-label={t('actions.moreFor', { username: user.username })}
          >
            {t('columns.actions')}
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <CreateEditUserButton user={user} menuItem />
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
    </div>
  )
}
