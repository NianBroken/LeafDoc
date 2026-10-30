// Avatar state belongs to the current session. Request generations prevent late
// status responses and previous accounts from overwriting completed mutations.
import * as avatarAPI from '@/shared/lib/api/avatar'
import { mapApiError } from '@/shared/lib/api/errors'
import i18next from '@/shared/lib/i18n'
import { useSessionStore } from '@/shared/state/session'
import { create } from 'zustand'

const t = (key: string) => i18next.t(key, { ns: 'settings' })
let requestGeneration = 0

type AvatarStore = {
  avatarVersion: number
  hasCustomAvatar: boolean | null
  isLoading: boolean
  isStatusLoading: boolean
  error: string | null
  statusError: string | null
  loadAvatarStatus: () => Promise<void>
  uploadAvatar: (file: File) => Promise<void>
  deleteAvatar: () => Promise<void>
}

export const useAvatarStore = create<AvatarStore>((set, get) => {
  // One mutation runs at a time; status reads never replace its result.
  const mutate = async (action: () => Promise<void>, custom: boolean) => {
    if (get().isLoading) return
    const generation = ++requestGeneration
    set({ isLoading: true, isStatusLoading: false, error: null })
    try {
      await action()
      if (generation === requestGeneration) {
        set({
          avatarVersion: Math.max(Date.now(), get().avatarVersion + 1),
          hasCustomAvatar: custom,
          statusError: null,
        })
      }
    } catch (err) {
      if (generation === requestGeneration) {
        set({
          error: mapApiError(
            err,
            t(
              custom
                ? 'account.avatar.uploadFailed'
                : 'account.avatar.deleteFailed',
            ),
          ).message,
        })
      }
      throw err
    } finally {
      if (generation === requestGeneration) set({ isLoading: false })
    }
  }

  return {
    avatarVersion: 0,
    hasCustomAvatar: null,
    isLoading: false,
    isStatusLoading: false,
    error: null,
    statusError: null,
    loadAvatarStatus: async () => {
      if (get().isLoading) return
      const generation = ++requestGeneration
      set({ isStatusLoading: true, statusError: null })
      try {
        const status = await avatarAPI.getAvatarStatus()
        if (generation === requestGeneration)
          set({ hasCustomAvatar: status.hasCustomAvatar })
      } catch (err) {
        if (generation === requestGeneration)
          set({
            statusError: mapApiError(err, t('account.avatar.loadFailed'))
              .message,
          })
      } finally {
        if (generation === requestGeneration) set({ isStatusLoading: false })
      }
    },
    uploadAvatar: (file) => mutate(() => avatarAPI.uploadAvatar(file), true),
    deleteAvatar: () => mutate(avatarAPI.deleteAvatar, false),
  }
})

// Session changes invalidate all pending work, including logout and re-login.
useSessionStore.subscribe((session, previous) => {
  if (session.user?.id === previous.user?.id) return
  requestGeneration++
  useAvatarStore.setState({
    avatarVersion: 0,
    hasCustomAvatar: null,
    isLoading: false,
    isStatusLoading: false,
    error: null,
    statusError: null,
  })
})
