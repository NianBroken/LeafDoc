// 本文件用真实管理页面验证加载状态、所属用户映射、撤销状态与确认框参数。
import '@/shared/lib/i18n'
import { DIALOG_DELETE_API_KEY_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useApiKeyStore } from '@/features/apikeys/state/apikeys'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useUserStore } from '@/features/users/state/users'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ApiKeysManagement from '@/features/apikeys/ApiKeysManagement'

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
import * as userAPI from '@/shared/lib/api/users'

describe('ApiKeysManagement', () => {
  // 清空两个列表和全局对话框，防止上一场景的数据影响加载与空列表判断。
  beforeEach(() => {
    useApiKeyStore.setState({ apiKeys: [] })
    useUserStore.setState({ users: [] })
    useDialogsStore.setState({ dialogType: null, dialogProps: null })
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows a loading state then the empty state when there are no keys', async () => {
    // 请求异步完成前展示加载文字，两个空结果完成后切换为空列表说明。
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue([])
    ;(userAPI.getUsers as ReturnType<typeof vi.fn>).mockResolvedValue([])

    render(<ApiKeysManagement />)

    expect(screen.getByText('Loading API keys...')).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.getByText('No API keys found.')).toBeInTheDocument(),
    )
  })

  it('renders the list of keys with the owning username resolved', async () => {
    // 密钥和用户通过相同 userId 关联，角色显示使用翻译而非原始小写值。
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue([
      {
        id: 'k1',
        name: 'agent key',
        userId: 'u1',
        prefix: 'ab12cd34',
        role: 'viewer',
        createdBy: 'admin1',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ])
    ;(userAPI.getUsers as ReturnType<typeof vi.fn>).mockResolvedValue([
      {
        id: 'u1',
        username: 'agent-owner',
        email: 'a@example.com',
        role: 'editor',
      },
    ])

    render(<ApiKeysManagement />)

    await waitFor(() =>
      expect(screen.getByText('agent key')).toBeInTheDocument(),
    )
    expect(screen.getByText('agent-owner')).toBeInTheDocument()
    expect(screen.getByText('Viewer')).toBeInTheDocument()
  })

  it('shows "Revoked" instead of the revoke action for an already-revoked key', async () => {
    // 撤销时间存在时使用状态文字，不向该行提供再次撤销的按钮。
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue([
      {
        id: 'k1',
        name: 'agent key',
        userId: 'u1',
        prefix: 'ab12cd34',
        role: 'viewer',
        createdBy: 'admin1',
        createdAt: '2026-01-01T00:00:00Z',
        revokedAt: '2026-01-02T00:00:00Z',
      },
    ])
    ;(userAPI.getUsers as ReturnType<typeof vi.fn>).mockResolvedValue([])

    render(<ApiKeysManagement />)

    await waitFor(() => expect(screen.getByText('Revoked')).toBeInTheDocument())
    expect(screen.queryByText('Revoke')).not.toBeInTheDocument()
  })

  it('clicking Revoke opens the delete confirmation dialog for that key', async () => {
    // 点击仅修改全局对话框状态，核对名称与标识均来自选中记录。
    const user = userEvent.setup()
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue([
      {
        id: 'k1',
        name: 'agent key',
        userId: 'u1',
        prefix: 'ab12cd34',
        role: 'viewer',
        createdBy: 'admin1',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ])
    ;(userAPI.getUsers as ReturnType<typeof vi.fn>).mockResolvedValue([])

    render(<ApiKeysManagement />)

    await waitFor(() =>
      expect(screen.getByText('agent key')).toBeInTheDocument(),
    )
    await user.click(screen.getByRole('button', { name: 'Revoke' }))

    expect(useDialogsStore.getState().dialogType).toBe(
      DIALOG_DELETE_API_KEY_CONFIRMATION,
    )
    expect(useDialogsStore.getState().dialogProps).toEqual({
      apiKeyId: 'k1',
      apiKeyName: 'agent key',
    })
  })
})
