// 检查本地密码输入框和凭据传递，接口失败由替身控制，不创建真实登录会话。
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { beforeEach, expect, it, vi } from 'vitest'
import { login } from '@/shared/lib/api/auth'
import { useSessionStore } from '@/shared/state/session'
import LoginForm from '@/features/auth/LoginForm'
vi.mock('@/shared/lib/api/auth', () => ({ login: vi.fn() }))
vi.mock('sonner', () => ({ toast: { error: vi.fn() } }))
beforeEach(() => {
  // 将用户状态归零并清除调用记录，使表单以未登录状态渲染。
  useSessionStore.setState({ user: null })
  vi.clearAllMocks()
})
it('shows a password login form', () => {
  // 路由上下文支持组件读取导航状态，此处仅断言密码输入框存在。
  render(
    <MemoryRouter>
      <LoginForm />
    </MemoryRouter>,
  )
  expect(screen.getByTestId('login-password')).toBeInTheDocument()
})
it('submits the entered credentials', async () => {
  // 使用拒绝结果避免成功跳转，断言关注输入值是否原样传入登录接口。
  vi.mocked(login).mockRejectedValue(new Error('test'))
  render(
    <MemoryRouter>
      <LoginForm />
    </MemoryRouter>,
  )
  await userEvent.type(screen.getByTestId('login-identifier'), 'NianBroken')
  await userEvent.type(screen.getByTestId('login-password'), 'Nian123456')
  await userEvent.click(screen.getByTestId('login-submit'))
  expect(login).toHaveBeenCalledWith('NianBroken', 'Nian123456')
})
