// 工具栏容器保存调用方编排的按钮列表，展示、权限判断和快捷键注册由使用方执行。
import { create } from 'zustand'

// 每个条目包含显示内容、交互状态和无参数动作，hotkey 是展示或注册所需的描述值。
// disabled、active 和 destructive 只作为数据保存，本容器不据此阻止 action 执行。
export interface ToolbarButton {
  id: string
  label: string
  tooltip?: string
  hotkey: string
  icon: React.ReactNode
  variant?: 'outline' | 'ghost' | 'link' | 'destructive' | 'default'
  className?: string
  destructive?: boolean
  disabled?: boolean
  active?: boolean
  action: () => void
}

export interface ToolbarState {
  buttons: ToolbarButton[]
  setButtons: (buttons: ToolbarButton[]) => void
  getButtons: () => ToolbarButton[]
}

// 设置动作整体替换数组，读取动作返回当前数组引用，不复制或重新排序按钮。
export const useToolbarStore = create<ToolbarState>((set, get) => ({
  buttons: [],
  setButtons: (buttons) => set({ buttons }),
  getButtons: () => get().buttons,
}))
