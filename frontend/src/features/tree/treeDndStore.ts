// 保存拖放界面的启用状态、当前拖动节点和当前落点，实际树数据仍由树状态容器维护。
import { create } from 'zustand'
import { DropTarget } from '@/features/tree/treeDndUtils'

type TreeDndState = {
  enabled: boolean
  activeId: string | null
  dropTarget: DropTarget | null
}

export const useTreeDndStore = create<TreeDndState>()(() => ({
  enabled: false,
  activeId: null,
  dropTarget: null,
}))
