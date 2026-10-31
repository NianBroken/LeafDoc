// 保存拖放界面的启用状态、当前拖动节点和当前落点，实际树数据仍由树状态容器维护。
import { create } from 'zustand'
import { DropTarget } from '@/features/tree/treeDndUtils'

type TreeDndState = {
  // enabled 是界面开关，activeId 是拖动节点，dropTarget 是拖动计算得出的落点。
  // 本容器只声明初值，具体动作由调用方使用状态容器接口写入。
  enabled: boolean
  activeId: string | null
  dropTarget: DropTarget | null
}

// 模块加载时拖放关闭，尚无拖动节点或落点，不把这些临时值持久化。
export const useTreeDndStore = create<TreeDndState>()(() => ({
  enabled: false,
  activeId: null,
  dropTarget: null,
}))
