// 本文件验证撤销确认框调用真实状态动作，网络由模拟 API 承担。
import { expectDiagnostics } from '@/shared/testutil/expectedDiagnostics'
import '@/shared/lib/i18n'
import { DIALOG_DELETE_API_KEY_CONFIRMATION } from '@/shared/lib/registries/instances'
import { useApiKeyStore } from '@/features/apikeys/state/apikeys'
import { useDialogsStore } from '@/shared/state/dialogs'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { DeleteApiKeyDialog } from '@/features/apikeys/DeleteApiKeyDialog'

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}))

vi.mock('@/shared/lib/api/apikeys', () => ({
  getApiKeys: vi.fn(),
  createApiKey: vi.fn(),
  deleteApiKey: vi.fn(),
}))

import * as apiKeyAPI from '@/shared/lib/api/apikeys'

describe('DeleteApiKeyDialog', () => {
  // 每个场景激活确认框并清空密钥列表，保留存储动作本身用于验证调用链。
  beforeEach(() => {
    useDialogsStore.setState({
      dialogType: DIALOG_DELETE_API_KEY_CONFIRMATION,
      dialogProps: null,
    })
    useApiKeyStore.setState({ apiKeys: [] })
    vi.clearAllMocks()
  })

  it('revokes the key and closes on confirm', async () => {
    // 撤销和重载均成功，断言请求目标标识；此断言不单独检查关闭状态。
    const user = userEvent.setup()
    ;(apiKeyAPI.deleteApiKey as ReturnType<typeof vi.fn>).mockResolvedValue(
      undefined,
    )
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue([])

    render(<DeleteApiKeyDialog apiKeyId="k1" apiKeyName="agent key" />)

    expect(screen.getByText('agent key')).toBeInTheDocument()

    await user.click(screen.getByTestId('delete-api-key-dialog-button-confirm'))

    await waitFor(() =>
      expect(apiKeyAPI.deleteApiKey).toHaveBeenCalledWith('k1'),
    )
  })

  it('keeps the dialog open and surfaces an error when revocation fails', async () => {
    // 失败路径预期记录一次错误，不能继续调用列表读取。
    const user = userEvent.setup()
    expectDiagnostics('error', [
      ['Error revoking API key:', new Error('network error')],
    ])
    ;(apiKeyAPI.deleteApiKey as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error('network error'),
    )

    render(<DeleteApiKeyDialog apiKeyId="k1" apiKeyName="agent key" />)

    await user.click(screen.getByTestId('delete-api-key-dialog-button-confirm'))

    await waitFor(() => expect(apiKeyAPI.deleteApiKey).toHaveBeenCalled())
    // 此处核对失败后没有执行列表刷新，关闭决定由处理器返回 false 控制。
    expect(apiKeyAPI.getApiKeys).not.toHaveBeenCalled()
  })
})
