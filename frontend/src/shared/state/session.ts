// 保存浏览器会话的用户显示信息与访问令牌到期时间，不在状态中保存令牌正文。
import type { SessionUser } from '@/shared/contracts/auth'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type UserInfo = SessionUser

// 时间戳以秒为单位，与后端 Unix 秒时间一致，不能直接当作毫秒时间传给 Date。
type UnixTimestampSeconds = number

// 刷新标志仅用于当前内存状态，clear 只清用户与到期时间，不重置刷新标志。
type SessionState = {
  // 该标志协调界面等待状态，不表示服务器刷新会话记录本身是否存在。
  isRefreshing: boolean
  accessTokenExpiresAt: UnixTimestampSeconds | null
  user: UserInfo | null
  setAccessTokenExpiresAt: (value: UnixTimestampSeconds | null) => void
  setUser: (user: UserInfo | null) => void
  setRefreshing: (value: boolean) => void
  clear: () => void
}

// persist 默认使用浏览器存储，只有 partialize 选出的两个字段参与持久化。
// 用户信息属于界面缓存，真实访问授权仍由服务端会话处理。
export const useSessionStore = create<SessionState>()(
  persist(
    (set) => ({
      user: null,
      isRefreshing: false,
      accessTokenExpiresAt: null,
      setAccessTokenExpiresAt: (value) => set({ accessTokenExpiresAt: value }),
      setRefreshing: (value) => set({ isRefreshing: value }),
      setUser: (user) => set({ user }),
      // 清除缓存身份和到期时间，调用方另行处理服务端退出及刷新状态。
      clear: () => set({ user: null, accessTokenExpiresAt: null }),
    }),
    {
      // 存储键固定用于同一站点下的会话缓存，服务端 Cookie 不由此中间件创建或删除。
      name: 'session-storage',
      // 方法和 isRefreshing 不写入存储，页面重新加载后刷新标志恢复初始值。
      partialize: (state) => ({
        accessTokenExpiresAt: state.accessTokenExpiresAt,
        user: state.user,
      }),
    },
  ),
)
