import { mapApiError } from '@/shared/lib/api/errors'
import { create } from 'zustand'
import i18next from '@/shared/lib/i18n'
import {
  fetchSnapshotStatus,
  fetchSnapshots,
  updateSnapshotSettings,
  triggerSnapshot,
  deleteSnapshot,
  SnapshotEntry,
  SnapshotSettings,
} from '@/shared/lib/api/snapshot'
import { sleep } from '@/shared/lib/sleep'

const POLL_INTERVAL_MS = 800
const POLL_ERROR_LIMIT = 3

interface SnapshotState {
  enabled: boolean
  retentionCount: number
  settings: SnapshotSettings | null
  isRunning: boolean
  lastSnapshotAt: string | null
  lastError: string
  lastPruneError: string
  snapshots: SnapshotEntry[]
  isLoading: boolean
  isListLoading: boolean
  statusError: string
  loadStatus: () => Promise<void>
  loadList: () => Promise<void>
  triggerNow: () => Promise<void>
  saveSettings: (settings: SnapshotSettings) => Promise<void>
  remove: (id: string) => Promise<void>
}

export const useSnapshotStore = create<SnapshotState>((set, get) => {
  // 创建请求只启动服务端任务，快照随后异步生成；轮询到状态不再运行后再刷新列表。
  async function pollUntilNotRunning(): Promise<void> {
    let consecutiveErrors = 0
    while (get().isRunning) {
      await sleep(POLL_INTERVAL_MS)
      try {
        const data = await fetchSnapshotStatus()
        set({
          isRunning: data.status?.isRunning ?? false,
          lastSnapshotAt: data.status?.lastSnapshotAt ?? null,
          lastError: data.status?.lastError ?? '',
          lastPruneError: data.status?.lastPruneError ?? '',
        })
        consecutiveErrors = 0
      } catch (err) {
        consecutiveErrors++
        if (consecutiveErrors >= POLL_ERROR_LIMIT) {
          set({
            statusError: mapApiError(
              err,
              i18next.t('statusLoadError', { ns: 'snapshot' }),
            ).message,
          })
          throw err
        }
      }
    }
  }

  return {
    enabled: false,
    retentionCount: 0,
    settings: null,
    isRunning: false,
    lastSnapshotAt: null,
    lastError: '',
    lastPruneError: '',
    snapshots: [],
    isLoading: false,
    isListLoading: false,
    statusError: '',

    loadStatus: async () => {
      set({ isLoading: true })
      try {
        const data = await fetchSnapshotStatus()
        set({
          enabled: data.enabled,
          retentionCount: data.retentionCount ?? 0,
          settings: data.settings ?? null,
          isRunning: data.status?.isRunning ?? false,
          lastSnapshotAt: data.status?.lastSnapshotAt ?? null,
          lastError: data.status?.lastError ?? '',
          lastPruneError: data.status?.lastPruneError ?? '',
          isLoading: false,
          statusError: '',
        })
      } catch (error) {
        console.error('Failed to load snapshot status', error)
        set({
          isLoading: false,
          statusError: mapApiError(
            error,
            i18next.t('statusLoadError', { ns: 'snapshot' }),
          ).message,
        })
      }
    },

    loadList: async () => {
      set({ isListLoading: true })
      try {
        const snapshots = await fetchSnapshots()
        set({ snapshots, isListLoading: false })
      } catch (error) {
        console.error('Failed to load snapshots', error)
        set({
          isListLoading: false,
          statusError: mapApiError(
            error,
            i18next.t('statusLoadError', { ns: 'snapshot' }),
          ).message,
        })
      }
    },

    triggerNow: async () => {
      let requestError: unknown
      let requestFailed = false
      try {
        await triggerSnapshot()
      } catch (err) {
        requestError = err
        requestFailed = true
      }
      // Refresh status even when the POST was rejected because another job is running.
      await get().loadStatus()
      let completionError: unknown
      let completionFailed = false
      try {
        await pollUntilNotRunning()
      } catch (err) {
        completionError = err
        completionFailed = true
      }
      await get().loadList()
      if (requestFailed) throw requestError
      if (completionFailed) throw completionError
      if (get().lastError) throw new Error(get().lastError)
      if (get().statusError) throw new Error(get().statusError)
    },

    saveSettings: async (settings) => {
      const saved = await updateSnapshotSettings(settings)
      set({ settings: saved, retentionCount: saved.retentionCount })
    },

    remove: async (id: string) => {
      await deleteSnapshot(id)
      await get().loadList()
    },
  }
})
