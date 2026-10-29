import { create } from 'zustand'
import i18next from '@/shared/lib/i18n'
import {
  triggerRestore,
  triggerRestoreFromUpload,
  getRestoreStatus,
  triggerSelfRestart,
  RestoreStatus,
} from '@/shared/lib/api/restore'
import { sleep } from '@/shared/lib/sleep'

const POLL_INTERVAL_MS = 800
const POLL_ERROR_LIMIT = 3

interface RestoreState {
  isLoading: boolean
  requiresLogin: boolean
  phase: string | null
  // The job receipt covers database replacement and all derived-state rebuilds.
  isResyncPhase: boolean
  needsIntervention: boolean
  versionWarning: string | null
  // Successful completion includes the server's derived-state rebuild.
  resyncConfirmed: boolean
  trigger: (id: string) => Promise<void>
  triggerUpload: (file: File) => Promise<void>
  selfRestart: () => Promise<void>
}

export const useRestoreStore = create<RestoreState>((set) => ({
  isLoading: false,
  requiresLogin: false,
  phase: null,
  isResyncPhase: false,
  needsIntervention: false,
  versionWarning: null,
  resyncConfirmed: true,

  trigger: async (id: string) => {
    set({
      isLoading: true,
      phase: null,
      isResyncPhase: false,
      needsIntervention: false,
      versionWarning: null,
      resyncConfirmed: true,
    })
    try {
      const receipt = await triggerRestore(id)
      await pollUntilDone(set, receipt.statusToken)
    } catch (err) {
      set({ isLoading: false })
      throw err
    }
  },

  triggerUpload: async (file: File) => {
    set({
      isLoading: true,
      phase: null,
      isResyncPhase: false,
      needsIntervention: false,
      versionWarning: null,
      resyncConfirmed: true,
    })
    try {
      const receipt = await triggerRestoreFromUpload(file)
      await pollUntilDone(set, receipt.statusToken)
    } catch (err) {
      set({ isLoading: false })
      throw err
    }
  },

  selfRestart: async () => {
    try {
      await triggerSelfRestart()
    } catch {
      // Expected: the connection drops when the server process replaces
      // itself mid-request. Not a real failure — see triggerSelfRestart's
      // doc comment in lib/api/restore.ts.
    }
  },
}))

// pollUntilDone polls the restore status endpoint until the job finishes,
// updating phase/versionWarning as it goes — shared by trigger(id) and
// triggerUpload(file) alike, since neither entrypoint affects how the
// server-side job reports progress from here on.
async function pollUntilDone(
  set: (partial: Partial<RestoreState>) => void,
  statusToken: string,
): Promise<void> {
  let consecutiveErrors = 0
  for (;;) {
    await sleep(POLL_INTERVAL_MS)

    let status: RestoreStatus
    try {
      status = await getRestoreStatus(statusToken)
      consecutiveErrors = 0
    } catch (err) {
      consecutiveErrors++
      if (consecutiveErrors >= POLL_ERROR_LIMIT) {
        set({ isLoading: false, phase: null })
        throw err
      }
      continue
    }

    set({
      phase: status.phase || null,
      versionWarning: status.versionWarning || null,
    })

    if (status.done) {
      if (status.needsIntervention) {
        set({ isLoading: false, needsIntervention: true })
        return
      }
      if (status.error) {
        set({ isLoading: false, phase: null })
        throw new Error(status.error)
      }
      set({
        isLoading: false,
        phase: null,
        isResyncPhase: false,
        requiresLogin: true,
      })
      return
    }
    if (!status.running) {
      set({ isLoading: false, phase: null })
      throw new Error(i18next.t('errors.jobLost', { ns: 'restore' }))
    }
  }
}
