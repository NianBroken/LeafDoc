// 共享状态只记录当前是否加载，进度数值、延迟显示和隐藏状态均由 Progressbar 组件维护。

import { create } from 'zustand'

export interface ProgressbarState {
  loading: boolean
  setLoading: (loading: boolean) => void
}

export const useProgressbarStore = create<ProgressbarState>((set) => ({
  loading: false,
  setLoading: (loading: boolean) => {
    // 直接替换布尔值，不对多个并行请求计数，调用方负责协调加载开始与结束。
    set({ loading })
  },
}))
