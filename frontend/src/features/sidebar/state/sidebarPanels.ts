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
      name: 'leafdoc-sidebar-panels', // 本地存储键，区分侧栏区块状态与其他界面偏好。
      merge: (persistedState, currentState) => {
        // 只接受完全由字符串组成的数组，空数组也有效；格式不符沿用内存默认值。
        // 不检查区块 ID 是否仍在注册表中，也不合并持久化对象的其他字段。
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
