import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { startSettingsSync } from '@/app/settingsSync'
import { toast } from 'sonner'

const { loadConfig, loadBranding, loadUsers } = vi.hoisted(() => ({
  loadConfig: vi.fn(),
  loadBranding: vi.fn(),
  loadUsers: vi.fn(),
}))
vi.mock('@/shared/state/config', () => ({
  useConfigStore: { getState: () => ({ loadConfig }) },
}))
vi.mock('@/features/branding/state/branding', () => ({
  useBrandingStore: { getState: () => ({ loadBranding }) },
}))
vi.mock('@/features/users/state/users', () => ({
  useUserStore: { getState: () => ({ loadUsers }) },
}))
vi.mock('@/shared/state/session', () => ({
  useSessionStore: {
    getState: () => ({ user: { role: 'admin', mustChangePassword: false } }),
  },
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn() } }))

describe('bounded settings synchronization', () => {
  let stop: (() => void) | undefined
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })
  afterEach(() => {
    stop?.()
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })
  const response = (cursor: string) => ({
    ok: true,
    json: async () => ({ cursor }),
  })
  const flush = async () => {
    for (let n = 0; n < 20; n++) await Promise.resolve()
  }

  it('refreshes changed cursors, waits through idle responses, and completes after disposal', async () => {
    let finish: (value: unknown) => void = () => {}
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(response('instance/1'))
      .mockResolvedValueOnce(response('instance/1'))
      .mockResolvedValueOnce(response('instance/2'))
      .mockImplementation(
        () =>
          new Promise((resolve) => {
            finish = resolve
          }),
      )
    vi.stubGlobal('fetch', fetchMock)
    stop = startSettingsSync()
    await flush()
    expect(fetchMock).toHaveBeenCalledTimes(4)
    expect(fetchMock.mock.calls[1][0]).toContain('cursor=instance%2F1')
    expect(fetchMock.mock.calls[0][1]).toEqual({
      credentials: 'include',
      keepalive: true,
    })
    expect(loadConfig).toHaveBeenCalledTimes(2)
    expect(loadBranding).toHaveBeenCalledTimes(2)
    expect(loadUsers).toHaveBeenCalledTimes(2)
    stop()
    finish(response('instance/3'))
    await flush()
    expect(fetchMock).toHaveBeenCalledTimes(4)
    expect(loadConfig).toHaveBeenCalledTimes(2)
    document.dispatchEvent(new Event('visibilitychange'))
    expect(loadConfig).toHaveBeenCalledTimes(2)
  })

  it('reports failures, retries with a delay, and cancels scheduled work on disposal', async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error('offline'))
    vi.stubGlobal('fetch', fetchMock)
    stop = startSettingsSync()
    await flush()
    expect(toast.error).toHaveBeenCalledExactlyOnceWith('offline')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(3000)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    stop()
    await vi.advanceTimersByTimeAsync(3000)
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it.each([
    [{ ok: false, status: 503 }, 'Settings sync: HTTP 503'],
    [{ ok: true, json: async () => ({}) }, 'Settings sync: invalid response'],
  ])('rejects unsuccessful or malformed responses', async (result, message) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(result))
    stop = startSettingsSync()
    await flush()
    expect(toast.error).toHaveBeenCalledExactlyOnceWith(message)
    expect(loadConfig).not.toHaveBeenCalled()
  })
})
