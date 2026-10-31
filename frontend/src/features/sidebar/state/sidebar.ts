// 保存侧栏显示状态、宽度及当前面板，状态通过本地存储持续保留。

import { SIDEBAR_TREE_PANEL_ID } from '@/shared/lib/registries/instances'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export const DEFAULT_SIDEBAR_WIDTH = 345
export const MIN_SIDEBAR_WIDTH = 220
export const MAX_SIDEBAR_WIDTH = 800

type SidebarStore = {
  sidebarMode: string
  setSidebarMode: (mode: string) => void

  sidebarVisible: boolean
  setSidebarVisible: (visible: boolean) => void

  sidebarWidth: number
  setSidebarWidth: (width: number) => void

  // 搜索面板已打开时再次按快捷键递增，用于通知面板重新聚焦输入框。
  searchFocusRequestId: number
  requestSearchFocus: () => void
}

export const useSidebarStore = create<SidebarStore>()(
  persist(
    (set) => ({
      sidebarMode: SIDEBAR_TREE_PANEL_ID,
      setSidebarMode: (mode) => set({ sidebarMode: mode }),

      sidebarVisible: true,
      setSidebarVisible: (visible) => set({ sidebarVisible: visible }),

      sidebarWidth: DEFAULT_SIDEBAR_WIDTH,
      setSidebarWidth: (width) => {
        const clamped = Math.min(
          MAX_SIDEBAR_WIDTH,
          Math.max(MIN_SIDEBAR_WIDTH, width),
        )
        set({ sidebarWidth: clamped })
      },

      searchFocusRequestId: 0,
      requestSearchFocus: () =>
        set((state) => ({
          searchFocusRequestId: state.searchFocusRequestId + 1,
        })),
    }),
    {
      name: 'leafdoc-sidebar', // localStorage key
      partialize: (state) => ({
        sidebarVisible: state.sidebarVisible,
        sidebarMode: state.sidebarMode,
        sidebarWidth: state.sidebarWidth,
      }),
    },
  ),
)
