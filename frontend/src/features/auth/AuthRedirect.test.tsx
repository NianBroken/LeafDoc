// 通过内存路由检查登录入口与受保护页面之间的跳转，认证接口使用替身返回确定结果。
// 页面探针直接展示路由状态，断言覆盖原路径保存、外部登录地址和成功登录后的目标页面。
import '@/shared/lib/i18n'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginForm from '@/features/auth/LoginForm'
import RequireAuth from '@/features/auth/RequireAuth'
import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'

const loginMock = vi.fn()

vi.mock('@/shared/lib/api/auth', () => ({
  login: (...args: unknown[]) => loginMock(...args),
}))

function LocationProbe() {
  // 只读取对象状态中的 redirectTo，缺少字段时显示空串，便于断言守卫传入的导航信息。
  const location = useLocation()
  const redirectTo =
    typeof location.state === 'object' &&
    location.state !== null &&
    'redirectTo' in location.state
      ? String((location.state as { redirectTo?: unknown }).redirectTo ?? '')
      : ''

  return (
    <>
      <div data-testid="pathname">{location.pathname}</div>
      <div data-testid="redirect-to">{redirectTo}</div>
    </>
  )
}

describe('Auth redirect flow', () => {
  beforeEach(() => {
    // 清空接口调用与会话，关闭远程用户认证，使各用例从未登录的本地认证入口开始。
    loginMock.mockReset()
    useConfigStore.setState({
      httpRemoteUserEnabled: false,
    })
    useSessionStore.setState({
      user: null,
      isRefreshing: false,
      accessTokenExpiresAt: null,
    })
  })

  it('preserves the requested path when redirecting to login', async () => {
    // 将路径、查询参数和页内锚点分别传给初始路由，检查守卫拼接后全部存入登录页状态。
    render(
      <MemoryRouter
        initialEntries={[
          {
            pathname: '/home-lab/services-ip-addresses',
            search: '?tab=ports',
            hash: '#ssh',
          },
        ]}
      >
        <Routes>
          <Route path="/login" element={<LocationProbe />} />
          <Route
            path="*"
            element={
              <RequireAuth>
                <div>Protected page</div>
              </RequireAuth>
            }
          />
        </Routes>
      </MemoryRouter>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('pathname')).toHaveTextContent('/login')
    })
    expect(screen.getByTestId('redirect-to')).toHaveTextContent(
      '/home-lab/services-ip-addresses?tab=ports#ssh',
    )
  })

  it('redirects externally instead of to /login when loginUrl is configured, preserving the requested path', async () => {
    // 用可写 href 观察整页跳转地址，避免测试环境实际离开当前文档。
    const originalLocation = window.location
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, href: '' },
    })

    try {
      useConfigStore.setState({
        loginUrl: 'https://idp.example.com/login',
      })

      render(
        <MemoryRouter initialEntries={['/some/protected/page']}>
          <Routes>
            <Route path="/login" element={<LocationProbe />} />
            <Route
              path="*"
              element={
                <RequireAuth>
                  <div>Protected page</div>
                </RequireAuth>
              }
            />
          </Routes>
        </MemoryRouter>,
      )

      const expectedReturnTo = encodeURIComponent(
        `${originalLocation.origin}/some/protected/page`,
      )
      await waitFor(() => {
        expect(window.location.href).toBe(
          `https://idp.example.com/login?redirect_uri=${expectedReturnTo}`,
        )
      })
      expect(screen.queryByTestId('pathname')).not.toBeInTheDocument()
    } finally {
      // 即使断言失败也还原浏览器位置对象，避免影响后续路由测试。
      Object.defineProperty(window, 'location', {
        configurable: true,
        value: originalLocation,
      })
    }
  })

  it('returns to the requested page after successful login', async () => {
    // 接口返回用户与令牌到期时间，表单据此建立会话，并使用路由状态中的目标导航。
    loginMock.mockResolvedValue({
      accessTokenExpiresAt: 1234567890,
      message: 'ok',
      user: {
        id: 'user-1',
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      },
    })

    const user = userEvent.setup()

    render(
      <MemoryRouter
        initialEntries={[
          {
            pathname: '/login',
            state: {
              redirectTo: '/home-lab/services-ip-addresses?tab=ports#ssh',
            },
          },
        ]}
      >
        <Routes>
          <Route path="/login" element={<LoginForm />} />
          <Route path="/" element={<div>Home page</div>} />
          <Route
            path="/home-lab/services-ip-addresses"
            element={<div>Requested page</div>}
          />
        </Routes>
      </MemoryRouter>,
    )

    await user.type(screen.getByTestId('login-identifier'), 'admin')
    await user.type(screen.getByTestId('login-password'), 'admin')
    await user.click(screen.getByTestId('login-submit'))

    await screen.findByText('Requested page')
    // 页面文本证明已离开登录页，接口参数另行验证提交的用户名和密码。
    expect(loginMock).toHaveBeenCalledWith('admin', 'admin')
  })
})
