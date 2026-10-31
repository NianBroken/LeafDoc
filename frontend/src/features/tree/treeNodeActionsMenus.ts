// 记录当前展开操作菜单的树节点 ID，空值表示没有节点菜单处于打开状态。

import { create } from 'zustand'

type TreeNodeActionsMenusStore = {
  openMenuNodeId: string | null
  setOpenMenuNodeId: (nodeId: string | null) => void
}

export const useTreeNodeActionsMenusStore = create<TreeNodeActionsMenusStore>(
  (set) => ({
    openMenuNodeId: null,
    setOpenMenuNodeId: (nodeId: string | null) => {
      // 菜单组件将目标节点 ID 写入共享状态，以保证任一时刻只匹配一个菜单。
      set({ openMenuNodeId: nodeId })
    },
  }),
)
