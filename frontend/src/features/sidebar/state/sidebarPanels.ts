// 记录侧栏各折叠区块是否展开，并将区块状态写入本地存储。

import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type SidebarPanelsStore = {
  openSections: string[]
  setOpenSections: (ids: string[]) => void
}

export const useSidebarPanelsStore = create<SidebarPanelsStore>()(
  persist(
    (set) => ({
      openSections: ['pinned', 'favorites', 'pages'],
      setOpenSections: (ids) => set({ openSections: ids }),
    }),
    {
      name: 'leafdoc-sidebar-panels', // localStorage key
      merge: (persistedState, currentState) => {
        const openSections = (
          persistedState as Partial<SidebarPanelsStore> | null | undefined
        )?.openSections

        return {
          ...currentState,
          openSections:
            Array.isArray(openSections) &&
            openSections.every((id) => typeof id === 'string')
              ? openSections
              : currentState.openSections,
        }
      },
    },
  ),
)
