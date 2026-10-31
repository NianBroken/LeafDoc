// 本文件通过请求替身检查全局偏好的默认值、写入内容与失败时的本地状态。
// 状态存储使用实际实现，请求不会发送到运行中的管理员接口。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  useUserSettingsStore,
  DEFAULT_PREFERENCES,
} from '@/shared/state/userSettings'
import { fetchWithAuth } from '@/shared/lib/api/client'
// 仅替换网络入口，各例自行提供成功的完整偏好或失败错误。
vi.mock('@/shared/lib/api/client', () => ({ fetchWithAuth: vi.fn() }))
describe('global preferences', () => {
  // 还原显示值并清空调用历史，不重新创建 Zustand 存储。
  beforeEach(() => {
    useUserSettingsStore.setState({ ...DEFAULT_PREFERENCES, loaded: false })
    vi.clearAllMocks()
  })
  // 集中默认配置必须给出中文、ISO 日期与二十四小时制。
  it('defaults to simplified Chinese and explicit date/time formats', () => {
    expect(useUserSettingsStore.getState()).toMatchObject({
      language: 'zh',
      dateFormat: 'iso',
      timeFormat: '24h',
    })
  })
  // 提交仅含 language 的补丁，服务端完整响应在请求完成后成为本地状态。
  it('writes global preferences and applies the authoritative response', async () => {
    vi.mocked(fetchWithAuth).mockResolvedValue({
      ...DEFAULT_PREFERENCES,
      language: 'de',
    })
    await useUserSettingsStore.getState().setLanguage('de')
    expect(fetchWithAuth).toHaveBeenCalledWith(
      '/api/admin/settings/preferences',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ language: 'de' }),
      }),
    )
    expect(useUserSettingsStore.getState().language).toBe('de')
  })
  // 请求拒绝必须传播给调用方，日期格式继续保持已提交的默认值。
  it('keeps committed state when saving fails', async () => {
    vi.mocked(fetchWithAuth).mockRejectedValue(new Error('unavailable'))
    await expect(
      useUserSettingsStore.getState().setDateFormat('dmy_dot'),
    ).rejects.toThrow()
    expect(useUserSettingsStore.getState().dateFormat).toBe('iso')
  })
})
