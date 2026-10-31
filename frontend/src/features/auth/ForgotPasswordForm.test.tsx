// 检查找回密码表单的请求参数和成功、失败反馈，测试不连接邮件服务。
import '@/shared/lib/i18n'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { toast } from 'sonner'
import ForgotPasswordForm from '@/features/auth/ForgotPasswordForm'

vi.mock('sonner', () => ({ toast: { error: vi.fn() } }))

const requestPasswordResetMock = vi.fn()

vi.mock('@/shared/lib/api/auth', () => ({
  requestPasswordReset: (...args: unknown[]) =>
    requestPasswordResetMock(...args),
}))

function renderForm() {
  // 为找回密码入口和返回登录入口提供内存路由，组件中的链接可使用真实路由上下文。
  return render(
    <MemoryRouter initialEntries={['/forgot-password']}>
      <Routes>
        <Route path="/forgot-password" element={<ForgotPasswordForm />} />
        <Route path="/login" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ForgotPasswordForm', () => {
  beforeEach(() => {
    // 每个用例独立设置接口结果，不沿用先前的调用记录或拒绝状态。
    requestPasswordResetMock.mockReset()
  })

  it('shows the same generic confirmation for a successful request', async () => {
    // 请求正常结束时显示统一确认文案，此断言不代表服务器已经发送或送达邮件。
    requestPasswordResetMock.mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderForm()

    await user.type(screen.getByTestId('forgot-password-identifier'), 'alice')
    await user.click(screen.getByTestId('forgot-password-submit'))

    await waitFor(() =>
      expect(screen.getByText('Check your email')).toBeInTheDocument(),
    )
    expect(requestPasswordResetMock).toHaveBeenCalledWith('alice')
  })

  // 接口拒绝时检查错误提示携带原始错误消息，不将请求失败当成成功确认。
  it('keeps the form and shows the actual request failure', async () => {
    requestPasswordResetMock.mockRejectedValue(new Error('boom'))
    const user = userEvent.setup()
    renderForm()

    await user.type(
      screen.getByTestId('forgot-password-identifier'),
      'no-such-user',
    )
    await user.click(screen.getByTestId('forgot-password-submit'))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith('boom'))
  })
})
