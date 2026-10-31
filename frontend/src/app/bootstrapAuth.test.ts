// 本文件以真实会话与配置存储配合模拟网络入口，验证身份初始化选择的请求和状态结果。
// 测试不发送服务器请求；等待断言用于观察 Hook 异步副作用完成后的共享状态。
import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import { fetchMe } from '@/shared/lib/api/auth'
import { ApiError, ensureRefresh } from '@/shared/lib/api/client'
import { useBootstrapAuth } from '@/app/bootstrapAuth'

// 保留认证模块的其余导出，仅替换当前身份查询，避免把待测 Hook 自身替换成模拟实现。
vi.mock('@/shared/lib/api/auth', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/lib/api/auth')>()
  return {
    ...actual,
    fetchMe: vi.fn(),
  }
})

// ApiError 仍使用原始类型，使认证失败分支执行真实的 instanceof 判断。
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

// 每例从普通会话模式和匿名缓存开始，具体案例单独设置需要验证的配置或身份。
beforeEach(() => {
  vi.clearAllMocks()
  ;(fetchMe as Mock).mockResolvedValue(null)
  ;(ensureRefresh as Mock).mockResolvedValue(undefined)
  useConfigStore.setState({
    hasLoaded: true,
    configLoadSucceeded: true,
    httpRemoteUserEnabled: false,
  })
  useSessionStore.setState({ user: null, accessTokenExpiresAt: null })
})

describe('useBootstrapAuth', () => {
  // 配置未成功读取时即使存在缓存用户，也先查询身份，不按未知模式决定刷新。
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

  // 远程用户模式把身份查询交给后端代理认证路径，不发送本地会话刷新请求。
  it('calls fetchMe when httpRemoteUserEnabled is true', async () => {
    useConfigStore.setState({
      configLoadSucceeded: true,
      httpRemoteUserEnabled: true,
    })

    renderHook(() => useBootstrapAuth(true))

    await waitFor(() => expect(fetchMe).toHaveBeenCalled())
    expect(ensureRefresh).not.toHaveBeenCalled()
  })

  // 普通认证模式已确认且缓存访问时间过期时，只调用刷新入口。
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

  // 没有缓存用户时查询匿名身份，空响应继续保持匿名，不凭空创建用户对象。
  it('reads anonymous identity without sending a token refresh request', async () => {
    renderHook(() => useBootstrapAuth(true))

    await waitFor(() => expect(fetchMe).toHaveBeenCalledOnce())
    expect(ensureRefresh).not.toHaveBeenCalled()
    expect(useSessionStore.getState().user).toBeNull()
  })

  // 将访问到期时间放在当前时间之后，断言有效会话通过身份查询确认而不是轮换刷新令牌。
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

  // 使用真实 ApiError 的 401 状态模拟认证拒绝，必须清空原缓存用户。
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

  // 普通 Error 不属于明确的客户端认证拒绝，初始化结束后保留缓存身份。
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
