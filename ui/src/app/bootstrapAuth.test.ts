import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import { fetchMe } from '@/shared/lib/api/auth'
import { ApiError, ensureRefresh } from '@/shared/lib/api/client'
import { useBootstrapAuth } from '@/app/bootstrapAuth'

vi.mock('@/shared/lib/api/auth', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/lib/api/auth')>()
  return {
    ...actual,
    fetchMe: vi.fn(),
  }
})

vi.mock('@/shared/lib/api/client', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@/shared/lib/api/client')>()
  return { ...actual, ensureRefresh: vi.fn() }
})

const testUser = {
  id: 'user-1',
  username: 'ivan',
  email: 'ivan@example.com',
  role: 'admin' as const,
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(fetchMe as Mock).mockResolvedValue(null)
  ;(ensureRefresh as Mock).mockResolvedValue(undefined)
  useConfigStore.setState({
    hasLoaded: true,
    configLoadSucceeded: true,
    httpRemoteUserEnabled: false,
    authDisabled: false,
  })
  useSessionStore.setState({ user: null, accessTokenExpiresAt: null })
})

describe('useBootstrapAuth', () => {
  it('calls fetchMe (not ensureRefresh) when config failed to load', async () => {
    useConfigStore.setState({
      configLoadSucceeded: false,
      httpRemoteUserEnabled: false,
    })
    useSessionStore.setState({ user: testUser })

    renderHook(() => useBootstrapAuth(true))

    await waitFor(() => expect(fetchMe).toHaveBeenCalled())
    expect(ensureRefresh).not.toHaveBeenCalled()
  })

  it('calls fetchMe when httpRemoteUserEnabled is true', async () => {
    useConfigStore.setState({
      configLoadSucceeded: true,
      httpRemoteUserEnabled: true,
    })

    renderHook(() => useBootstrapAuth(true))

    await waitFor(() => expect(fetchMe).toHaveBeenCalled())
    expect(ensureRefresh).not.toHaveBeenCalled()
  })

  it('refreshes an expired recorded session in a confirmed session-auth deployment', async () => {
    useConfigStore.setState({
      configLoadSucceeded: true,
      httpRemoteUserEnabled: false,
    })
    useSessionStore.setState({ user: testUser, accessTokenExpiresAt: 1 })

    renderHook(() => useBootstrapAuth(true))

    await waitFor(() => expect(ensureRefresh).toHaveBeenCalled())
    expect(fetchMe).not.toHaveBeenCalled()
  })

  it('reads anonymous identity without sending a token refresh request', async () => {
    renderHook(() => useBootstrapAuth(true))

    await waitFor(() => expect(fetchMe).toHaveBeenCalledOnce())
    expect(ensureRefresh).not.toHaveBeenCalled()
    expect(useSessionStore.getState().user).toBeNull()
  })

  it('confirms a current session without rotating its refresh token on navigation', async () => {
    useSessionStore.setState({
      user: testUser,
      accessTokenExpiresAt: Math.floor(Date.now() / 1000) + 300,
    })
    ;(fetchMe as Mock).mockResolvedValue(testUser)

    renderHook(() => useBootstrapAuth(true))

    await waitFor(() => expect(fetchMe).toHaveBeenCalledOnce())
    expect(ensureRefresh).not.toHaveBeenCalled()
    expect(useSessionStore.getState().user).toEqual(testUser)
  })

  it('clears the user when fetchMe fails with an auth error', async () => {
    useConfigStore.setState({
      configLoadSucceeded: true,
      httpRemoteUserEnabled: true,
    })
    useSessionStore.setState({ user: testUser })
    ;(fetchMe as Mock).mockRejectedValue(new ApiError('unauthorized', 401))

    renderHook(() => useBootstrapAuth(true))

    await waitFor(() => expect(useSessionStore.getState().user).toBeNull())
  })

  it('preserves the user when fetchMe fails with a server/network error', async () => {
    useConfigStore.setState({
      configLoadSucceeded: true,
      httpRemoteUserEnabled: true,
    })
    useSessionStore.setState({ user: testUser })
    ;(fetchMe as Mock).mockRejectedValue(new Error('network error'))

    renderHook(() => useBootstrapAuth(true))

    await waitFor(() =>
      expect(useSessionStore.getState().isRefreshing).toBe(false),
    )
    expect(useSessionStore.getState().user).toEqual(testUser)
  })
})
