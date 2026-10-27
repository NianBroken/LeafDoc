import type { SessionUser } from '@/shared/contracts/auth'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type UserInfo = SessionUser

// Unix timestamp in seconds, matching the backend's use of time.Now().Unix().
type UnixTimestampSeconds = number

type SessionState = {
  isRefreshing: boolean
  accessTokenExpiresAt: UnixTimestampSeconds | null
  user: UserInfo | null
  setAccessTokenExpiresAt: (value: UnixTimestampSeconds | null) => void
  setUser: (user: UserInfo | null) => void
  setRefreshing: (value: boolean) => void
  clear: () => void
}

export const useSessionStore = create<SessionState>()(
  persist(
    (set) => ({
      user: null,
      isRefreshing: false,
      accessTokenExpiresAt: null,
      setAccessTokenExpiresAt: (value) => set({ accessTokenExpiresAt: value }),
      setRefreshing: (value) => set({ isRefreshing: value }),
      setUser: (user) => set({ user }),
      clear: () => set({ user: null, accessTokenExpiresAt: null }),
    }),
    {
      name: 'session-storage',
      partialize: (state) => ({
        accessTokenExpiresAt: state.accessTokenExpiresAt,
        user: state.user,
      }),
    },
  ),
)
