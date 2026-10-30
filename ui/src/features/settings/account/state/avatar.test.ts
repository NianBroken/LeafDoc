import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/lib/api/avatar', () => ({
  uploadAvatar: vi.fn(),
  deleteAvatar: vi.fn(),
  avatarUrl: vi.fn(),
  getAvatarStatus: vi.fn(),
}))

import * as avatarAPI from '@/shared/lib/api/avatar'
import { useAvatarStore } from '@/features/settings/account/state/avatar'
import { useSessionStore } from '@/shared/state/session'

function makeFile(name = 'avatar.png'): File {
  return new File(['fake-bytes'], name, { type: 'image/png' })
}

describe('useAvatarStore', () => {
  beforeEach(() => {
    useSessionStore.getState().setUser({
      id: crypto.randomUUID(),
      username: 'test',
      email: '',
      role: 'viewer',
    })
    useAvatarStore.setState({
      avatarVersion: 0,
      isLoading: false,
      error: null,
      hasCustomAvatar: null,
      isStatusLoading: false,
      statusError: null,
    })
    vi.mocked(avatarAPI.getAvatarStatus).mockReset()
    vi.mocked(avatarAPI.uploadAvatar).mockReset()
    vi.mocked(avatarAPI.deleteAvatar).mockReset()
  })

  it('uploadAvatar bumps avatarVersion on success', async () => {
    vi.mocked(avatarAPI.uploadAvatar).mockResolvedValue(undefined)

    await useAvatarStore.getState().uploadAvatar(makeFile())

    const state = useAvatarStore.getState()
    expect(state.avatarVersion).toBeGreaterThan(0)
    expect(state.isLoading).toBe(false)
    expect(state.error).toBeNull()
    expect(state.hasCustomAvatar).toBe(true)
  })

  it('uploadAvatar sets an error and rethrows when the API call fails', async () => {
    vi.mocked(avatarAPI.uploadAvatar).mockRejectedValue(new Error('boom'))

    await expect(
      useAvatarStore.getState().uploadAvatar(makeFile()),
    ).rejects.toThrow('boom')

    const state = useAvatarStore.getState()
    expect(state.error).toBe('boom')
    expect(state.isLoading).toBe(false)
    // No prior client state to roll back to — version stays untouched.
    expect(state.avatarVersion).toBe(0)
  })

  it('deleteAvatar bumps avatarVersion on success', async () => {
    vi.mocked(avatarAPI.deleteAvatar).mockResolvedValue(undefined)

    await useAvatarStore.getState().deleteAvatar()

    const state = useAvatarStore.getState()
    expect(state.avatarVersion).toBeGreaterThan(0)
    expect(state.isLoading).toBe(false)
    expect(state.error).toBeNull()
    expect(state.hasCustomAvatar).toBe(false)
  })

  it('deleteAvatar sets an error and rethrows when the API call fails', async () => {
    vi.mocked(avatarAPI.deleteAvatar).mockRejectedValue(new Error('boom'))

    await expect(useAvatarStore.getState().deleteAvatar()).rejects.toThrow(
      'boom',
    )

    const state = useAvatarStore.getState()
    expect(state.error).toBe('boom')
    expect(state.isLoading).toBe(false)
    expect(state.avatarVersion).toBe(0)
  })

  it('isLoading is true while the upload is in flight', async () => {
    let resolveUpload: () => void = () => {}
    vi.mocked(avatarAPI.uploadAvatar).mockReturnValue(
      new Promise((resolve) => {
        resolveUpload = () => resolve(undefined)
      }),
    )

    const promise = useAvatarStore.getState().uploadAvatar(makeFile())
    expect(useAvatarStore.getState().isLoading).toBe(true)

    resolveUpload()
    await promise
    expect(useAvatarStore.getState().isLoading).toBe(false)
  })
})

// Deferred responses exercise the ordering that occurs across uploads and sessions.
describe('avatar status ordering', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useSessionStore.getState().setUser({
      id: crypto.randomUUID(),
      username: 'test',
      email: '',
      role: 'viewer',
    })
  })

  it('loads default and custom state from the server', async () => {
    vi.mocked(avatarAPI.getAvatarStatus).mockResolvedValue({
      hasCustomAvatar: false,
    })
    await useAvatarStore.getState().loadAvatarStatus()
    expect(useAvatarStore.getState().hasCustomAvatar).toBe(false)
    vi.mocked(avatarAPI.getAvatarStatus).mockResolvedValue({
      hasCustomAvatar: true,
    })
    await useAvatarStore.getState().loadAvatarStatus()
    expect(useAvatarStore.getState().hasCustomAvatar).toBe(true)
  })

  it('retains unknown status on failure and allows retry', async () => {
    vi.mocked(avatarAPI.getAvatarStatus).mockRejectedValueOnce(
      new Error('offline'),
    )
    await useAvatarStore.getState().loadAvatarStatus()
    expect(useAvatarStore.getState().statusError).toBe('offline')
    expect(useAvatarStore.getState().hasCustomAvatar).toBeNull()
    expect(useAvatarStore.getState().isStatusLoading).toBe(false)
    vi.mocked(avatarAPI.getAvatarStatus).mockResolvedValue({
      hasCustomAvatar: false,
    })
    await useAvatarStore.getState().loadAvatarStatus()
    expect(useAvatarStore.getState().statusError).toBeNull()
  })

  it('ignores a status response that predates a completed upload', async () => {
    let resolveStatus!: (value: { hasCustomAvatar: boolean }) => void
    vi.mocked(avatarAPI.getAvatarStatus).mockReturnValue(
      new Promise((resolve) => {
        resolveStatus = resolve
      }),
    )
    const loading = useAvatarStore.getState().loadAvatarStatus()
    vi.mocked(avatarAPI.uploadAvatar).mockResolvedValue(undefined)
    await useAvatarStore.getState().uploadAvatar(makeFile())
    resolveStatus({ hasCustomAvatar: false })
    await loading
    expect(useAvatarStore.getState().hasCustomAvatar).toBe(true)
  })

  it('clears state on account switch and ignores the old response', async () => {
    let resolveStatus!: (value: { hasCustomAvatar: boolean }) => void
    vi.mocked(avatarAPI.getAvatarStatus).mockReturnValue(
      new Promise((resolve) => {
        resolveStatus = resolve
      }),
    )
    const loading = useAvatarStore.getState().loadAvatarStatus()
    useSessionStore.getState().clear()
    resolveStatus({ hasCustomAvatar: true })
    await loading
    expect(useAvatarStore.getState().hasCustomAvatar).toBeNull()
    expect(useAvatarStore.getState().isStatusLoading).toBe(false)
  })

  it('does not apply an upload result after logout', async () => {
    let resolveUpload!: () => void
    vi.mocked(avatarAPI.uploadAvatar).mockReturnValue(
      new Promise((resolve) => {
        resolveUpload = resolve
      }),
    )
    const uploading = useAvatarStore.getState().uploadAvatar(makeFile())
    useSessionStore.getState().clear()
    resolveUpload()
    await uploading
    expect(useAvatarStore.getState().hasCustomAvatar).toBeNull()
    expect(useAvatarStore.getState().avatarVersion).toBe(0)
  })

  it('prevents duplicate mutations and retains the custom avatar after delete failure', async () => {
    let rejectDelete!: (error: Error) => void
    useAvatarStore.setState({ hasCustomAvatar: true })
    vi.mocked(avatarAPI.deleteAvatar).mockReturnValue(
      new Promise((_, reject) => {
        rejectDelete = reject
      }),
    )
    const deleting = useAvatarStore.getState().deleteAvatar()
    await useAvatarStore.getState().deleteAvatar()
    expect(avatarAPI.deleteAvatar).toHaveBeenCalledTimes(1)
    rejectDelete(new Error('failed'))
    await expect(deleting).rejects.toThrow('failed')
    expect(useAvatarStore.getState().hasCustomAvatar).toBe(true)
  })
})
