// 在当前页面进程内保存最近的 Wiki 位置，供导航与设置页面返回流程共享。
// 本存储没有持久化中间件，浏览器整页重载后位置恢复为 null。
// 路径是否应被记录由调用方判断，此文件原样保存 pathname、search 和导航 state。

import { create } from 'zustand'

// state 保留导航状态，其中的访问标识可供返回页面时恢复对应访问的滚动位置。
export type StoredWikiLocation = {
  pathname: string
  search: string
  state: unknown
}

type LastWikiLocationStore = {
  location: StoredWikiLocation | null
  setLocation: (location: StoredWikiLocation | null) => void
}

// setLocation 直接保存对象引用，传 null 清除位置，不在这里执行导航。
export const useLastWikiLocationStore = create<LastWikiLocationStore>(
  (set) => ({
    location: null,
    setLocation: (location) => set({ location }),
  }),
)
