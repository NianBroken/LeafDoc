// 此存储保存密钥公开元数据，创建结果中的明文令牌只返回调用方，不写入列表状态。
import * as apiKeyAPI from '@/shared/lib/api/apikeys'
import { create } from 'zustand'

// 所有异步动作向调用方传播失败，由页面或对话框决定错误展示和关闭行为。
type ApiKeyStore = {
  apiKeys: apiKeyAPI.ApiKey[]
  reset: () => void
  loadApiKeys: () => Promise<void>
  createApiKey: (
    data: Parameters<typeof apiKeyAPI.createApiKey>[0],
  ) => Promise<apiKeyAPI.CreateApiKeyResult>
  deleteApiKey: (id: string) => Promise<void>
}

// 使用 Zustand 共享当前列表，写操作成功后重新请求完整列表，不做本地推测更新。
export const useApiKeyStore = create<ApiKeyStore>((set, get) => ({
  apiKeys: [],

  // 清空已有列表，不取消已经在途的网络请求。
  reset: () => set({ apiKeys: [] }),

  loadApiKeys: async () => {
    // 请求成功才整体替换，读取失败时保留原列表并向外抛出错误。
    const apiKeys = await apiKeyAPI.getApiKeys()
    set({ apiKeys })
  },

  createApiKey: async (data) => {
    // 创建与刷新依次执行，刷新失败也会让整个动作拒绝，令牌结果不会返回调用方。
    const result = await apiKeyAPI.createApiKey(data)
    await get().loadApiKeys()
    return result
  },

  deleteApiKey: async (id) => {
    // 后端撤销成功后刷新列表，使撤销标记和其他元数据以服务器返回值为准。
    await apiKeyAPI.deleteApiKey(id)
    await get().loadApiKeys()
  },
}))
