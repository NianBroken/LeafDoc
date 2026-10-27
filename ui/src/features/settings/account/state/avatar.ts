// Self-service profile-picture state owns upload/delete actions and cache invalidation.
// Errors remain available to the calling view, and loading state ends on every path.

import * as avatarAPI from '@/shared/lib/api/avatar'
import i18next from '@/shared/lib/i18n'
import { create } from 'zustand'

const t = (key: string) => i18next.t(key, { ns: 'settings' })

type AvatarStore = {
  avatarVersion: number
  isLoading: boolean
  error: string | null

  uploadAvatar: (file: File) => Promise<void>
  deleteAvatar: () => Promise<void>
}

export const useAvatarStore = create<AvatarStore>((set) => ({
  avatarVersion: 0,
  isLoading: false,
  error: null,

  uploadAvatar: async (file) => {
    set({ isLoading: true, error: null })
    try {
      await avatarAPI.uploadAvatar(file)
      set({ avatarVersion: Date.now() })
    } catch (err) {
      set({
        error:
          err instanceof Error ? err.message : t('account.avatar.uploadFailed'),
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },

  deleteAvatar: async () => {
    set({ isLoading: true, error: null })
    try {
      await avatarAPI.deleteAvatar()
      set({ avatarVersion: Date.now() })
    } catch (err) {
      set({
        error:
          err instanceof Error ? err.message : t('account.avatar.deleteFailed'),
      })
      throw err
    } finally {
      set({ isLoading: false })
    }
  },
}))
