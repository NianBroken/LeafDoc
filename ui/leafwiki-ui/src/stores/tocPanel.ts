// The viewer owns the lifetime of temporary TOC choices; no global setting is written here.
import { create } from 'zustand'
type Store = {
  collapsed: boolean
  setCollapsed: (value: boolean) => void
  toggleCollapsed: () => void
}
export const useTocPanelStore = create<Store>((set) => ({
  collapsed: false,
  setCollapsed: (collapsed) => set({ collapsed }),
  toggleCollapsed: () => set((s) => ({ collapsed: !s.collapsed })),
}))
