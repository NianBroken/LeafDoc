// 使用实际菜单、快捷键处理器和对话框管理器检查入口协作，路由哨兵记录页面切换。
// 全局注册负责提供对话框内容，浏览器整页跳转通过可写位置对象观察。
import '@/app/registrations'
import { DialogManager } from '@/app/components/DialogManager'
import { HotKeyHandler } from '@/app/components/HotKeyHandler'
import UserMenu from '@/features/auth/components/UserMenu'
import * as authAPI from '@/shared/lib/api/auth'
import { useConfigStore } from '@/shared/state/config'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useSessionStore } from '@/shared/state/session'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

describe('UserMenu', () => {
  afterEach(() => {
    // 还原通过测试工具设置的全局对象；位置对象由对应测试组单独恢复。
    vi.unstubAllGlobals()
  })

  beforeEach(() => {
    // 清空对话框并建立本地编辑者会话，各角色与外部跳转用例再覆盖所需状态。
    useDialogsStore.setState({
      dialogType: null,
      dialogProps: null,
    })
    useConfigStore.setState({
      httpRemoteUserEnabled: false,
      loginUrl: '',
      logoutUrl: '',
    })
    useSessionStore.setState({
      user: {
        id: 'user-1',
        username: 'alice',
        email: 'alice@example.com',
        role: 'editor',
      },
    })
  })

  it('navigates to /settings when the Settings item is clicked', async () => {
    // 点击头像按钮再点击设置项，用目标路由的哨兵元素确认导航结果。
    const user = userEvent.setup({ delay: null })

    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<UserMenu />} />
          <Route
            path="/settings"
            element={<div data-testid="settings-page-sentinel">SETTINGS</div>}
          />
        </Routes>
      </MemoryRouter>,
    )

    const avatar = screen.getByTestId('user-menu-avatar')
    await user.click(avatar.closest('button') as HTMLButtonElement)
    await user.click(screen.getByTestId('user-menu-settings'))

    await waitFor(() => {
      expect(screen.getByTestId('settings-page-sentinel')).toBeInTheDocument()
    })
  })

  it('opens the shortcuts dialog from the user menu', async () => {
    // 菜单中不展示版本信息与快捷键组合，打开帮助后才检查说明内容与组合键文本。
    const user = userEvent.setup({ delay: null })

    render(
      <MemoryRouter>
        <UserMenu />
        <DialogManager />
      </MemoryRouter>,
    )

    const avatar = screen.getByTestId('user-menu-avatar')
    const trigger = avatar.closest('button')

    expect(trigger).toBeTruthy()
    await user.click(trigger as HTMLButtonElement)

    expect(screen.queryByText(/test-version|Version /)).not.toBeInTheDocument()
    expect(screen.queryByText('Ctrl+/')).not.toBeInTheDocument()
    const menuItem = await screen.findByText('Keyboard Shortcuts')
    await user.click(menuItem)

    await waitFor(() => {
      expect(screen.getByTestId('shortcuts-help-dialog')).toBeInTheDocument()
    })
    expect(screen.getByText('Available keyboard shortcuts')).toBeInTheDocument()
    expect(screen.getByText('Go to page')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+/')).toBeInTheDocument()
  })

  it('opens the shortcuts dialog via the keyboard shortcut', async () => {
    // 向窗口发送 Ctrl+/，由快捷键处理器触发菜单注册的帮助命令。
    render(
      <MemoryRouter>
        <UserMenu />
        <HotKeyHandler />
        <DialogManager />
      </MemoryRouter>,
    )

    fireEvent.keyDown(window, { key: '/', code: 'Slash', ctrlKey: true })

    await waitFor(() => {
      expect(screen.getByTestId('shortcuts-help-dialog')).toBeInTheDocument()
    })
  })

  it('does not open editing shortcuts for anonymous visitors', async () => {
    // 匿名状态仍挂载处理器与管理器，断言菜单没有注册可打开编辑帮助的入口。
    useSessionStore.setState({ user: null })

    render(
      <MemoryRouter>
        <UserMenu />
        <HotKeyHandler />
        <DialogManager />
      </MemoryRouter>,
    )

    fireEvent.keyDown(window, { key: '/', code: 'Slash', ctrlKey: true })

    await waitFor(() => {
      expect(
        screen.queryByTestId('shortcuts-help-dialog'),
      ).not.toBeInTheDocument()
    })
  })

  describe('viewer role', () => {
    // 只读角色保留设置入口，但不展示编辑快捷键菜单，也不注册对应键盘命令。
    beforeEach(() => {
      useSessionStore.setState({
        user: {
          id: 'user-2',
          username: 'bob',
          email: 'bob@example.com',
          role: 'viewer',
        },
      })
    })

    it('does not show the keyboard shortcuts menu item for viewer role', async () => {
      const user = userEvent.setup({ delay: null })

      render(
        <MemoryRouter>
          <UserMenu />
        </MemoryRouter>,
      )

      const avatar = screen.getByTestId('user-menu-avatar')
      const trigger = avatar.closest('button')
      await user.click(trigger as HTMLButtonElement)

      expect(screen.queryByText('Keyboard Shortcuts')).not.toBeInTheDocument()
    })

    it('still shows the Settings item for viewer role', async () => {
      const user = userEvent.setup({ delay: null })

      render(
        <MemoryRouter>
          <UserMenu />
        </MemoryRouter>,
      )

      const avatar = screen.getByTestId('user-menu-avatar')
      await user.click(avatar.closest('button') as HTMLButtonElement)

      expect(screen.getByTestId('user-menu-settings')).toBeInTheDocument()
    })

    it('does not open the shortcuts dialog via keyboard shortcut for viewer role', async () => {
      render(
        <MemoryRouter>
          <UserMenu />
          <HotKeyHandler />
          <DialogManager />
        </MemoryRouter>,
      )

      fireEvent.keyDown(window, { key: '/', code: 'Slash', ctrlKey: true })

      await new Promise((r) => setTimeout(r, 100))
      // 等待已发送的键盘事件处理后再检查对话框不存在。
      expect(
        screen.queryByTestId('shortcuts-help-dialog'),
      ).not.toBeInTheDocument()
    })
  })

  describe('logout redirect', () => {
    // 本地登录页使用独立哨兵，区分站内导航与外部 href 赋值。
    const renderWithLoginRoute = () =>
      render(
        <MemoryRouter initialEntries={['/']}>
          <Routes>
            <Route path="/" element={<UserMenu />} />
            <Route
              path="/login"
              element={<div data-testid="login-form-sentinel">LOGIN</div>}
            />
          </Routes>
        </MemoryRouter>,
      )

    let originalLocation: Location

    beforeEach(() => {
      originalLocation = window.location
      Object.defineProperty(window, 'location', {
        configurable: true,
        value: { ...originalLocation, href: '' },
      })
    })

    afterEach(() => {
      // 还原原始位置对象，使后续用例继续使用浏览器测试环境的位置属性。
      Object.defineProperty(window, 'location', {
        configurable: true,
        value: originalLocation,
      })
    })

    it('redirects straight to the logout URL without rendering the local login screen', async () => {
      // 登出接口由替身直接完成，检查菜单调用接口后采用配置的整页退出地址。
      const user = userEvent.setup({ delay: null })
      const logoutSpy = vi.spyOn(authAPI, 'logout').mockResolvedValue()
      useConfigStore.setState({
        logoutUrl: 'https://control-plane.example.com/logout',
      })

      renderWithLoginRoute()

      const avatar = screen.getByTestId('user-menu-avatar')
      await user.click(avatar.closest('button') as HTMLButtonElement)
      await user.click(screen.getByTestId('user-menu-logout'))

      expect(logoutSpy).toHaveBeenCalled()
      expect(window.location.href).toBe(
        'https://control-plane.example.com/logout',
      )
      expect(
        screen.queryByTestId('login-form-sentinel'),
      ).not.toBeInTheDocument()
    })

    it('falls back to the local /login route when no logout URL is configured', async () => {
      // 未配置外部退出地址时等待站内登录页出现，确认菜单执行本地路由导航。
      const user = userEvent.setup({ delay: null })
      useConfigStore.setState({ logoutUrl: '' })

      renderWithLoginRoute()

      const avatar = screen.getByTestId('user-menu-avatar')
      await user.click(avatar.closest('button') as HTMLButtonElement)
      await user.click(screen.getByTestId('user-menu-logout'))

      await waitFor(() => {
        expect(screen.getByTestId('login-form-sentinel')).toBeInTheDocument()
      })
    })
  })

  describe('login redirect', () => {
    // 匿名用户显示登录按钮，配置外部入口时点击直接设置对应地址。
    let originalLocation: Location

    beforeEach(() => {
      originalLocation = window.location
      Object.defineProperty(window, 'location', {
        configurable: true,
        value: { ...originalLocation, href: '' },
      })
      useSessionStore.setState({ user: null })
    })

    afterEach(() => {
      Object.defineProperty(window, 'location', {
        configurable: true,
        value: originalLocation,
      })
    })

    it('redirects to the login URL when clicking Login and loginUrl is configured', () => {
      useConfigStore.setState({
        loginUrl: 'https://idp.example.com/login',
      })

      render(
        <MemoryRouter>
          <UserMenu />
        </MemoryRouter>,
      )

      fireEvent.click(screen.getByRole('button', { name: 'Login' }))

      expect(window.location.href).toBe('https://idp.example.com/login')
    })
  })
})
