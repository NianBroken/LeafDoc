// 页面链接状态查询容器保存最近结果、加载标志与错误文本，不维护按页面分开的缓存。
import { fetchLinkStatus, type LinkStatusResult } from '@/shared/lib/api/links'
import i18next from '@/shared/lib/i18n'
import { create } from 'zustand'

type LinkStatusStore = {
  status: LinkStatusResult | null
  loading: boolean
  error: string | null
  fetchLinkStatusForPage: (pageId: string) => Promise<void>
  clear: () => void
}

export const useLinkStatusStore = create<LinkStatusStore>((set) => ({
  status: null,
  loading: false,
  error: null,

  // 清空只重置本地字段，不取消已发出的请求，后续响应仍能写回状态。
  clear: () => set({ status: null, loading: false, error: null }),

  fetchLinkStatusForPage: async (pageId: string) => {
    // 空字符串立即清空结果并记录本地化错误，不进行网络请求，也不裁剪非空 ID。
    if (!pageId) {
      set({
        status: null,
        loading: false,
        error: i18next.t('backlinks.pageIdRequired', { ns: 'viewer' }),
      })
      return
    }
    // 请求开始保留旧结果，成功整体替换；失败只写错误并结束加载，旧结果仍在。
    // 并发调用没有序号或取消机制，完成较晚的响应会成为当前状态。
    set({ loading: true, error: null })
    try {
      const data = await fetchLinkStatus(pageId)
      set({ status: data, loading: false })
    } catch (err: unknown) {
      const msg =
        err instanceof Error
          ? err.message
          : i18next.t('backlinks.fetchFailedFallback', { ns: 'viewer' })
      set({ error: msg, loading: false })
    }
  },
}))
