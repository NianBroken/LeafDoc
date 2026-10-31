// 本文件独立验证密钥存储的列表读取、清空和写入后重载，不挂载页面组件。
import type { ApiKey, CreateApiKeyResult } from '@/shared/lib/api/apikeys'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useApiKeyStore } from '@/features/apikeys/state/apikeys'

vi.mock('@/shared/lib/api/apikeys', () => ({
  getApiKeys: vi.fn(),
  createApiKey: vi.fn(),
  deleteApiKey: vi.fn(),
}))

import * as apiKeyAPI from '@/shared/lib/api/apikeys'

// makeKey 提供完整元数据样本，覆盖值放在末尾，场景可指定不同标识或状态。
const makeKey = (overrides: Partial<ApiKey> = {}): ApiKey => ({
  id: 'k1',
  name: 'agent key',
  userId: 'u1',
  prefix: 'ab12cd34',
  role: 'viewer',
  createdBy: 'admin1',
  createdAt: '2026-01-01T00:00:00Z',
  ...overrides,
})

describe('useApiKeyStore', () => {
  beforeEach(() => {
    useApiKeyStore.setState({ apiKeys: [] })
    vi.clearAllMocks()
  })

  it('loadApiKeys populates state from the API', async () => {
    // 成功读取后以服务器数组替换状态，不在存储中重新生成记录。
    const keys = [makeKey()]
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue(keys)

    await useApiKeyStore.getState().loadApiKeys()

    expect(useApiKeyStore.getState().apiKeys).toEqual(keys)
  })

  it('reset clears the list', () => {
    // 预填一条记录后调用同步清空动作，核对列表恢复为空数组。
    useApiKeyStore.setState({ apiKeys: [makeKey()] })
    useApiKeyStore.getState().reset()
    expect(useApiKeyStore.getState().apiKeys).toEqual([])
  })

  it('createApiKey returns the secret and refreshes the list', async () => {
    // 创建结果包含明文令牌，列表重载只包含公开元数据，二者分别断言。
    const created = makeKey({ id: 'k2' })
    const result: CreateApiKeyResult = {
      key: created,
      secret: 'lw_ab12cd34_secret',
    }
    ;(apiKeyAPI.createApiKey as ReturnType<typeof vi.fn>).mockResolvedValue(
      result,
    )
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue([
      created,
    ])

    const returned = await useApiKeyStore
      .getState()
      .createApiKey({ name: 'agent key', userId: 'u1' })

    expect(returned).toEqual(result)
    expect(apiKeyAPI.getApiKeys).toHaveBeenCalledTimes(1)
    expect(useApiKeyStore.getState().apiKeys).toEqual([created])
  })

  it('deleteApiKey removes the key and refreshes the list', async () => {
    // 模拟服务器撤销后返回空列表，确认提交标识与重载后的最终状态。
    useApiKeyStore.setState({ apiKeys: [makeKey()] })
    ;(apiKeyAPI.deleteApiKey as ReturnType<typeof vi.fn>).mockResolvedValue(
      undefined,
    )
    ;(apiKeyAPI.getApiKeys as ReturnType<typeof vi.fn>).mockResolvedValue([])

    await useApiKeyStore.getState().deleteApiKey('k1')

    expect(apiKeyAPI.deleteApiKey).toHaveBeenCalledWith('k1')
    expect(useApiKeyStore.getState().apiKeys).toEqual([])
  })
})
