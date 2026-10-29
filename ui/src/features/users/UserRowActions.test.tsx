import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { UserRowActions } from './UserRowActions'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useSessionStore } from '@/shared/state/session'
import {
  DIALOG_CHANGE_USER_PASSWORD,
  DIALOG_USER_FORM,
} from '@/shared/lib/registries/instances'
import type { User } from '@/shared/lib/api/users'

const managedUser: User = {
  id: 'managed-user',
  username: 'writer',
  email: 'writer@example.com',
  role: 'editor',
  mustSetPassword: false,
}

beforeEach(() => {
  useDialogsStore.getState().closeDialog()
  useSessionStore.getState().setUser(null)
})

afterEach(cleanup)

// Text triggers retain menu keyboard behavior and the existing dialog actions.
describe('UserRowActions', () => {
  it('opens with the keyboard and edits the selected user', async () => {
    const user = userEvent.setup()
    render(<UserRowActions user={managedUser} />)
    const trigger = screen.getByRole('button', {
      name: 'More actions for writer',
    })
    expect(trigger).toHaveTextContent('Actions')
    expect(trigger.querySelector('svg')).toBeNull()
    await user.tab()
    await user.keyboard('{Enter}')
    const edit = await screen.findByRole('menuitem', { name: 'Edit' })
    await user.click(edit)
    expect(useDialogsStore.getState().dialogType).toBe(DIALOG_USER_FORM)
    expect(useDialogsStore.getState().dialogProps).toEqual({
      user: managedUser,
    })
  })

  it('retains password actions for other users and hides them for oneself', async () => {
    const user = userEvent.setup()
    const { unmount } = render(<UserRowActions user={managedUser} />)
    await user.click(screen.getByRole('button'))
    await user.click(
      await screen.findByRole('menuitem', { name: 'Change Password' }),
    )
    expect(useDialogsStore.getState().dialogType).toBe(
      DIALOG_CHANGE_USER_PASSWORD,
    )
    expect(useDialogsStore.getState().dialogProps).toEqual({
      userId: managedUser.id,
      username: managedUser.username,
    })
    unmount()
    useSessionStore.getState().setUser(managedUser)
    render(<UserRowActions user={managedUser} />)
    await user.click(screen.getByRole('button'))
    await screen.findByRole('menuitem', { name: 'Edit' })
    expect(
      screen.queryByRole('menuitem', { name: 'Change Password' }),
    ).toBeNull()
    expect(screen.queryByRole('menuitem', { name: 'Delete' })).toBeNull()
  })
})
