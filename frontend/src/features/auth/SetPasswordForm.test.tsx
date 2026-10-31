// 检查重置密码与接受邀请共用表单的校验、接口选择和成功页面，认证请求全部使用替身。
import '@/shared/lib/i18n'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { toast } from 'sonner'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SetPasswordForm } from '@/features/auth/SetPasswordForm'

const confirmPasswordResetMock = vi.fn()
const acceptInviteMock = vi.fn()

vi.mock('@/shared/lib/api/auth', () => ({
  confirmPasswordReset: (...args: unknown[]) =>
    confirmPasswordResetMock(...args),
  acceptInvite: (...args: unknown[]) => acceptInviteMock(...args),
}))

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}))

// 字段校验错误包含字段名和消息数组，表单通过字段错误处理器显示对应消息。
// 此对象作为接口拒绝值使用，测试检查字段消息显示，并排除直接展示错误分类字符串。
const validationErrorBody = {
  error: 'validation_error',
  fields: [
    {
      field: 'newPassword',
      message: 'New password must be at least 8 characters long',
    },
  ],
}

function renderResetForm(path = '/reset-password?token=abc.def') {
  // 默认地址带有待提交令牌，登录哨兵页面用于验证缺少令牌时的跳转。
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/reset-password"
          element={<SetPasswordForm mode="reset" />}
        />
        <Route path="/login" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

function renderInviteForm(path = '/accept-invite?token=abc.def') {
  // 邀请模式的成功目标为首页，首页哨兵文本用于确认实际完成导航。
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/accept-invite"
          element={<SetPasswordForm mode="invite" />}
        />
        <Route path="/" element={<div>Home page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('SetPasswordForm reset mode', () => {
  // 重置与邀请请求分别清空，防止共享表单在错误模式调用接口而被遗漏。
  beforeEach(() => {
    confirmPasswordResetMock.mockReset()
    acceptInviteMock.mockReset()
    vi.mocked(toast.error).mockReset()
  })

  it('redirects to /login when no token is present in the URL', () => {
    renderResetForm('/reset-password')
    expect(screen.getByText('Login page')).toBeInTheDocument()
  })

  it('rejects mismatched passwords without calling the API', async () => {
    // 两个值均满足长度门槛，单独触发确认密码不一致的本地校验分支。
    const user = userEvent.setup()
    renderResetForm()

    await user.type(screen.getByTestId('set-password-new'), 'password-one')
    await user.type(screen.getByTestId('set-password-confirm'), 'password-two')
    await user.click(screen.getByTestId('set-password-submit'))

    expect(screen.getByText('Passwords do not match')).toBeInTheDocument()
    expect(confirmPasswordResetMock).not.toHaveBeenCalled()
  })

  it('rejects a too-short password inline without calling the API', async () => {
    // 两次输入相同短密码，验证长度错误显示在表单内且不会发出请求。
    const user = userEvent.setup()
    renderResetForm()

    await user.type(screen.getByTestId('set-password-new'), 'short')
    await user.type(screen.getByTestId('set-password-confirm'), 'short')
    await user.click(screen.getByTestId('set-password-submit'))

    expect(
      screen.getByText('Password must be at least 8 characters long'),
    ).toBeInTheDocument()
    expect(confirmPasswordResetMock).not.toHaveBeenCalled()
  })

  it('surfaces a readable message (not "validation_error") when the API rejects the password', async () => {
    confirmPasswordResetMock.mockRejectedValue(validationErrorBody)
    const user = userEvent.setup()
    renderResetForm()

    await user.type(screen.getByTestId('set-password-new'), 'a-new-password')
    await user.type(
      screen.getByTestId('set-password-confirm'),
      'a-new-password',
    )
    await user.click(screen.getByTestId('set-password-submit'))

    await waitFor(() => expect(confirmPasswordResetMock).toHaveBeenCalledOnce())
    // 字段错误处理器原样显示接口返回的字段消息，错误分类值不作为面向用户的提示。
    expect(
      await screen.findByText(validationErrorBody.fields[0].message),
    ).toBeInTheDocument()
    expect(screen.queryByText('validation_error')).not.toBeInTheDocument()
    expect(toast.error).not.toHaveBeenCalledWith('validation_error')
  })

  it('shows a success message and does NOT navigate home on success', async () => {
    // 接口成功后断言令牌与密码参数以及确认文案，此用例没有配置首页路由。
    confirmPasswordResetMock.mockResolvedValue({
      user: {
        id: 'u1',
        username: 'alice',
        email: 'a@x.com',
        role: 'editor',
      },
    })
    const user = userEvent.setup()
    renderResetForm()

    await user.type(screen.getByTestId('set-password-new'), 'a-new-password')
    await user.type(
      screen.getByTestId('set-password-confirm'),
      'a-new-password',
    )
    await user.click(screen.getByTestId('set-password-submit'))

    await waitFor(() =>
      expect(confirmPasswordResetMock).toHaveBeenCalledWith(
        'abc.def',
        'a-new-password',
      ),
    )
    expect(await screen.findByText('Password reset')).toBeInTheDocument()
  })
})

describe('SetPasswordForm invite mode', () => {
  // 同一输入界面在邀请模式调用 acceptInvite，成功结果包含会话数据。
  beforeEach(() => {
    confirmPasswordResetMock.mockReset()
    acceptInviteMock.mockReset()
    vi.mocked(toast.error).mockReset()
  })

  it('surfaces a readable message (not "validation_error") when the API rejects the password', async () => {
    // 邀请接口也使用相同字段错误结构，验证共享错误处理覆盖该入口。
    acceptInviteMock.mockRejectedValue(validationErrorBody)
    const user = userEvent.setup()
    renderInviteForm()

    await user.type(screen.getByTestId('set-password-new'), 'a-new-password')
    await user.type(
      screen.getByTestId('set-password-confirm'),
      'a-new-password',
    )
    await user.click(screen.getByTestId('set-password-submit'))

    await waitFor(() => expect(acceptInviteMock).toHaveBeenCalledOnce())
    expect(
      await screen.findByText(validationErrorBody.fields[0].message),
    ).toBeInTheDocument()
    expect(screen.queryByText('validation_error')).not.toBeInTheDocument()
  })

  it('calls acceptInvite and navigates home on success', async () => {
    // 先返回用户与令牌到期时间，再通过首页文本检查邀请成功后的页面切换。
    acceptInviteMock.mockResolvedValue({
      accessTokenExpiresAt: 123,
      message: 'ok',
      user: {
        id: 'u1',
        username: 'bob',
        email: 'b@x.com',
        role: 'viewer',
      },
    })
    const user = userEvent.setup()
    renderInviteForm()

    await user.type(screen.getByTestId('set-password-new'), 'a-new-password')
    await user.type(
      screen.getByTestId('set-password-confirm'),
      'a-new-password',
    )
    await user.click(screen.getByTestId('set-password-submit'))

    await waitFor(() =>
      expect(acceptInviteMock).toHaveBeenCalledWith(
        'abc.def',
        'a-new-password',
      ),
    )
    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })
})
