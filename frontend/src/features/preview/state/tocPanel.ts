// 保存当前查看器目录面板折叠状态，不把临时选择写入全局设置或持久化存储。
import { create } from 'zustand'
type Store = {
  collapsed: boolean
  setCollapsed: (value: boolean) => void
  toggleCollapsed: () => void
}
export const useTocPanelStore = create<Store>((set) => ({
  // 面板默认展开，设置动作直接采用调用方传入值，翻转动作按当前状态取反。
  collapsed: false,
  setCollapsed: (collapsed) => set({ collapsed }),
  toggleCollapsed: () => set((s) => ({ collapsed: !s.collapsed })),
}))
