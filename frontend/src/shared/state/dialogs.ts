// 保存全局当前对话框类型和传入属性，实际组件由对话框管理器按类型选择。
// 同时只有一个类型槽位，打开其他类型会替换当前内容，不维护对话框栈。

import { create } from 'zustand'

// 类型名使用字符串，属性不在存储层校验，具体对话框负责解释传入字段。
type DialogsStore = {
  dialogType: string | null
  dialogProps: Record<string, unknown> | null
  openDialog: (
    dialogType: string,
    dialogProps?: Record<string, unknown>,
  ) => void
  isAnyDialogOpen: () => boolean
  closeDialog: () => void
}

export const useDialogsStore = create<DialogsStore>((set, get) => ({
  dialogType: null,
  dialogProps: null,
  // 原样设置类型和可选属性，未传属性时运行时保存 undefined。
  openDialog: (dialogType: string, dialogProps?: Record<string, unknown>) => {
    set({ dialogType, dialogProps })
  },
  // 只以类型是否为 null 判断打开状态，不检查组件是否已经渲染。
  isAnyDialogOpen: () => {
    return get().dialogType !== null
  },
  // 同时清除类型和属性，避免下一个对话框复用先前的属性引用。
  closeDialog: () => {
    set({ dialogType: null, dialogProps: null })
  },
}))
