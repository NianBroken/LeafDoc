// The panel exposes deletion only after a successful custom-avatar status read.
import '@/shared/lib/i18n'
import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AvatarPanel } from './AvatarPanel'
import { useSessionStore } from '@/shared/state/session'
import { useAvatarStore } from './state/avatar'
import * as avatarAPI from '@/shared/lib/api/avatar'
vi.mock('@/shared/lib/api/avatar', () => ({
  getAvatarStatus: vi.fn(),
  uploadAvatar: vi.fn(),
  deleteAvatar: vi.fn(),
  avatarUrl: () => '/avatars/test',
}))
beforeEach(() => {
  vi.clearAllMocks()
  useSessionStore.getState().setUser({
    id: crypto.randomUUID(),
    username: 'test',
    email: '',
    role: 'viewer',
  })
})
describe('AvatarPanel', () => {
  it('hides removal for a default avatar', async () => {
    vi.mocked(avatarAPI.getAvatarStatus).mockResolvedValue({
      hasCustomAvatar: false,
    })
    render(<AvatarPanel />)
    await waitFor(() =>
      expect(useAvatarStore.getState().hasCustomAvatar).toBe(false),
    )
    expect(
      screen.queryByRole('button', { name: 'Remove picture' }),
    ).not.toBeInTheDocument()
  })
  it('shows removal for a custom avatar', async () => {
    vi.mocked(avatarAPI.getAvatarStatus).mockResolvedValue({
      hasCustomAvatar: true,
    })
    render(<AvatarPanel />)
    expect(
      await screen.findByRole('button', { name: 'Remove picture' }),
    ).toBeInTheDocument()
  })
  it('offers a retry without offering deletion on status failure', async () => {
    vi.mocked(avatarAPI.getAvatarStatus).mockRejectedValue(new Error('offline'))
    render(<AvatarPanel />)
    expect(await screen.findByRole('alert')).toHaveTextContent('offline')
    expect(
      screen.getByRole('button', { name: 'Retry picture status' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Remove picture' }),
    ).not.toBeInTheDocument()
  })
})
