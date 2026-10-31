/**
 * 保存浅色、深色或跟随系统的用户选择，组件通过统一状态动作切换主题。
 * 初始化读取本地存储，没有非空值时采用 system；实际浅深色由主题解析钩子计算。
 * 类型断言只约束编译检查，此处没有校验本地存储字符串是否属于三种声明值。
 */

import { create } from 'zustand'

type DesignModeStore = {
  mode: 'light' | 'dark' | 'system'
  setMode: (mode: 'light' | 'dark' | 'system') => void
}

export const useDesignModeStore = create<DesignModeStore>((set) => ({
  mode:
    (localStorage.getItem('design-mode') as 'light' | 'dark' | 'system') ||
    'system',

  setMode: (mode) => {
    // 先写本地存储，再通知订阅者；存储写入抛出异常时不会执行后面的状态更新。
    localStorage.setItem('design-mode', mode)
    set({ mode })
  },
}))
