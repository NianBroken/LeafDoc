// 本文件用真实表单和存储验证提交条件、令牌展示、剪贴板、日期转换和字段错误。
// 网络与提示组件被替换，测试不创建服务器密钥。
import { expectDiagnostics } from '@/shared/testutil/expectedDiagnostics'
import '@/shared/lib/i18n'
import { DIALOG_API_KEY_FORM } from '@/shared/lib/registries/instances'
import { useApiKeyStore } from '@/features/apikeys/state/apikeys'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useUserStore } from '@/features/users/state/users'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiKeyFormDialog } from '@/features/apikeys/ApiKeyFormDialog'

// 为 Radix Select 使用的指针与滚动接口提供测试环境实现，避免模拟 DOM 缺失方法。
beforeAll(() => {
  Element.prototype.hasPointerCapture = () => false
  Element.prototype.releasePointerCapture = () => {}
  Element.prototype.scrollIntoView = () => {}
})

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}))

vi.mock('@/shared/lib/api/apikeys', () => ({
  getApiKeys: vi.fn(),
  createApiKey: vi.fn(),
  deleteApiKey: vi.fn(),
}))

vi.mock('@/shared/lib/api/users', () => ({
  getUsers: vi.fn(),
  createUser: vi.fn(),
  updateUser: vi.fn(),
  deleteUser: vi.fn(),
}))

import * as apiKeyAPI from '@/shared/lib/api/apikeys'

describe('ApiKeyFormDialog', () => {
  // 每个场景打开同一种对话框并提供一个所属用户，清除上个场景的状态和调用记录。
  beforeEach(() => {
    useDialogsStore.setState({
      dialogType: DIALOG_API_KEY_FORM,
      dialogProps: null,
    })
    useApiKeyStore.setState({ apiKeys: [] })
    useUserStore.setState({
      users: [
        {
          id: 'u1',
          username: 'agent-owner',
          email: 'a@example.com',
          role: 'editor',
          mustSetPassword: false,
        },
      ],
    })
    vi.clearAllMocks()
  })

  it('disables Create until a name and an owner are set', async () => {
    // 分别检查初始、只有名称、名称与所属用户均已填写三个阶段。
    const user = userEvent.setup()
    render(<ApiKeyFormDialog />)

    expect(
      screen.getByTestId('api-key-form-dialog-button-confirm'),
    ).toBeDisabled()

    await user.type(
      screen.getByPlaceholderText('e.g. research-agent'),
      'agent key',
    )
    expect(
      screen.getByTestId('api-key-form-dialog-button-confirm'),
    ).toBeDisabled()

    await user.click(screen.getByTestId('api-key-owner-select'))
    await user.click(await screen.findByRole('option', { name: 'agent-owner' }))

    expect(
      screen.getByTestId('api-key-form-dialog-button-confirm'),
    ).not.toBeDisabled()
  })

  it('reveals the one-time secret after creation and copies it', async () => {
    const user = userEvent.setup()
    // userEvent.setup 会设置剪贴板替身，这里在其后安装可断言的写入函数。
    const clipboardWriteText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: clipboardWriteText },
      configurable: true,
    })
    ;(apiKeyAPI.createApiKey as ReturnType<typeof vi.fn>).mockResolvedValue({
      key: {
        id: 'k1',
        name: 'agent key',
        userId: 'u1',
        prefix: 'ab12cd34',
        role: 'viewer',
        createdBy: 'admin1',
        createdAt: '2026-01-01T00:00:00Z',
      },
      secret: 'lw_ab12cd34_supersecret',
    })
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue([])

    render(<ApiKeyFormDialog />)

    await user.type(
      screen.getByPlaceholderText('e.g. research-agent'),
      'agent key',
    )
    await user.click(screen.getByTestId('api-key-owner-select'))
    await user.click(await screen.findByRole('option', { name: 'agent-owner' }))
    await user.click(screen.getByTestId('api-key-form-dialog-button-confirm'))

    await waitFor(() =>
      expect(screen.getByTestId('api-key-secret')).toHaveValue(
        'lw_ab12cd34_supersecret',
      ),
    )
    // 表单请求携带名称和用户但省略角色，后端按空角色采用查看者权限。
    expect(apiKeyAPI.createApiKey).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'agent key', userId: 'u1' }),
    )
    expect(apiKeyAPI.createApiKey).not.toHaveBeenCalledWith(
      expect.objectContaining({ role: expect.anything() }),
    )

    await user.click(screen.getByTestId('api-key-secret-copy'))
    expect(clipboardWriteText).toHaveBeenCalledWith('lw_ab12cd34_supersecret')

    // 令牌展示状态不再提供创建确认按钮，避免同一表单继续提交。
    expect(
      screen.queryByTestId('api-key-form-dialog-button-confirm'),
    ).not.toBeInTheDocument()
  })

  it('normalizes a same-day expiry to the end of the local day', async () => {
    // 预期值按运行环境本地时区构造当日结束时刻，再转为 UTC。
    // 断言提交的字符串而非固定偏移，测试不向后端验证日期是否在未来。
    const expectedExpiresAt = new Date('2026-07-07T23:59:59').toISOString()
    const user = userEvent.setup()
    ;(apiKeyAPI.createApiKey as ReturnType<typeof vi.fn>).mockResolvedValue({
      key: {
        id: 'k1',
        name: 'agent key',
        userId: 'u1',
        prefix: 'ab12cd34',
        role: 'viewer',
        createdBy: 'admin1',
        createdAt: '2026-01-01T00:00:00Z',
      },
      secret: 'lw_ab12cd34_supersecret',
    })

    render(<ApiKeyFormDialog />)

    await user.type(
      screen.getByPlaceholderText('e.g. research-agent'),
      'agent key',
    )
    await user.click(screen.getByTestId('api-key-owner-select'))
    await user.click(await screen.findByRole('option', { name: 'agent-owner' }))
    await user.type(screen.getByTestId('api-key-expires-at'), '2026-07-07')
    await user.click(screen.getByTestId('api-key-form-dialog-button-confirm'))

    await waitFor(() =>
      expect(apiKeyAPI.createApiKey).toHaveBeenCalledWith(
        expect.objectContaining({ expiresAt: expectedExpiresAt }),
      ),
    )
  })

  it('surfaces field validation errors from the API without closing the dialog', async () => {
    // 登记预期告警并让请求返回字段错误，确认名称下方显示后端文字。
    const user = userEvent.setup()
    expectDiagnostics('warn', [
      [
        {
          error: 'validation_error',
          fields: [{ field: 'name', message: 'Name must not be empty' }],
        },
      ],
    ])
    ;(apiKeyAPI.createApiKey as ReturnType<typeof vi.fn>).mockRejectedValue({
      error: 'validation_error',
      fields: [{ field: 'name', message: 'Name must not be empty' }],
    })

    render(<ApiKeyFormDialog />)

    await user.type(screen.getByPlaceholderText('e.g. research-agent'), 'x')
    await user.click(screen.getByTestId('api-key-owner-select'))
    await user.click(await screen.findByRole('option', { name: 'agent-owner' }))
    await user.click(screen.getByTestId('api-key-form-dialog-button-confirm'))

    await waitFor(() =>
      expect(screen.getByText('Name must not be empty')).toBeInTheDocument(),
    )
    // 提交失败没有进入令牌展示分支，输入表单仍供用户纠正字段。
    expect(screen.queryByTestId('api-key-secret')).not.toBeInTheDocument()
  })
})
