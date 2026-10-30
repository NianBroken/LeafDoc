import * as importAPI from '@/shared/lib/api/import'
import { mapApiError } from '@/shared/lib/api/errors'
import i18next from '@/shared/lib/i18n'
import { sleep } from '@/shared/lib/sleep'
import { toast } from 'sonner'
import { create } from 'zustand'
import { useTreeStore } from '@/features/tree/state/tree'

const t = (key: string, opts?: Record<string, unknown>) =>
  i18next.t(key, { ...opts, ns: 'importer' })

type ImportStore = {
  creatingImportPlan: boolean
  executingImportPlan: boolean
  cancelingImportPlan: boolean
  loadingImportPlan: boolean
  importPlan: importAPI.ImportPlan | null
  importResult: importAPI.ImportResult | null
  createImportPlan: (sourcePath: File) => Promise<boolean>
  loadImportPlan: () => Promise<void>
  executeImportPlan: () => Promise<void>
  cancelImportPlan: () => Promise<boolean>
}

const IMPORT_POLL_INTERVAL_MS = 1000
const IMPORT_POLL_RETRY_LIMIT = 3

async function pollImportPlanUntilSettled(
  initialPlan: importAPI.ImportPlan,
  set: (partial: Partial<ImportStore>) => void,
): Promise<importAPI.ImportPlan> {
  let currentPlan = initialPlan
  let consecutivePollErrors = 0

  while (currentPlan.execution_status === 'running') {
    await sleep(IMPORT_POLL_INTERVAL_MS)

    try {
      const nextPlan = await importAPI.getImportPlan()
      if (!nextPlan) throw new Error(t('toast.loadPlanErrorFallback'))
      currentPlan = nextPlan
      consecutivePollErrors = 0
      set({
        importPlan: currentPlan,
        importResult: currentPlan.execution_result ?? null,
      })
    } catch (err) {
      consecutivePollErrors++
      if (consecutivePollErrors >= IMPORT_POLL_RETRY_LIMIT) {
        throw err
      }
    }
  }

  return currentPlan
}

export const useImportStore = create<ImportStore>((set, get) => ({
  importPlan: null,
  creatingImportPlan: false,
  executingImportPlan: false,
  cancelingImportPlan: false,
  loadingImportPlan: false,
  importResult: null,
  createImportPlan: async (sourcePath: File) => {
    set({ creatingImportPlan: true })
    try {
      const importPlan = await importAPI.createImportPlanFromZip(sourcePath)
      toast.success(t('toast.planCreatedSuccess'))
      set({ importPlan, importResult: null })
      return true
    } catch (err) {
      toast.error(mapApiError(err, t('toast.createPlanErrorFallback')).message)
      return false
    } finally {
      set({ creatingImportPlan: false })
    }
  },
  loadImportPlan: async () => {
    set({ loadingImportPlan: true })
    try {
      let importPlan = await importAPI.getImportPlan()
      if (!importPlan) {
        set({ importPlan: null, importResult: null })
        return
      }
      set({
        importPlan,
        importResult: importPlan.execution_result ?? null,
      })

      if (importPlan.execution_status === 'running') {
        set({ executingImportPlan: true })
        importPlan = await pollImportPlanUntilSettled(importPlan, set)
        set({
          importPlan,
          importResult: importPlan.execution_result ?? null,
        })
      }
    } catch (err) {
      toast.error(mapApiError(err, t('toast.loadPlanErrorFallback')).message)
      return
    } finally {
      set({ loadingImportPlan: false, executingImportPlan: false })
    }
  },
  executeImportPlan: async () => {
    const importPlan = get().importPlan
    if (importPlan === null) {
      toast.error(t('toast.noPlanToExecute'))
      return
    }
    try {
      set({ executingImportPlan: true, importResult: null })
      let currentPlan = await importAPI.executeImportPlan()
      set({ importPlan: currentPlan, importResult: null })

      currentPlan = await pollImportPlanUntilSettled(currentPlan, set)

      if (currentPlan.execution_status === 'completed') {
        toast.success(t('toast.executeSuccess'))
        set({
          importPlan: currentPlan,
          importResult: currentPlan.execution_result ?? null,
        })
      } else if (currentPlan.execution_status === 'canceled') {
        toast.success(t('toast.canceledSuccess'))
        set({
          importPlan: currentPlan,
          importResult: currentPlan.execution_result ?? null,
        })
      } else if (currentPlan.execution_status === 'failed') {
        set({ importPlan: currentPlan })
        throw new Error(
          currentPlan.execution_error || t('toast.executionFailedFallback'),
        )
      }
    } catch (err) {
      toast.error(mapApiError(err, t('toast.executeErrorFallback')).message)
    } finally {
      set({ executingImportPlan: false })
      // reload tree
      useTreeStore.getState().reloadTree()
    }
  },
  cancelImportPlan: async () => {
    const importPlan = get().importPlan
    if (importPlan === null) {
      toast.error(t('toast.noPlanToClear'))
      return false
    }
    try {
      set({ cancelingImportPlan: true })
      const response = await importAPI.cancelImportPlan()

      if (
        response &&
        response.execution_status === 'running' &&
        response.cancel_requested
      ) {
        set({ importPlan: response })
        const finalPlan = await pollImportPlanUntilSettled(response, set)
        toast.success(
          finalPlan.execution_status === 'canceled'
            ? t('toast.canceledSuccess')
            : t('toast.canceledBeforeCompletion'),
        )
        set({
          importPlan: finalPlan,
          importResult: finalPlan.execution_result ?? null,
        })
        useTreeStore.getState().reloadTree()
        return finalPlan.execution_status === 'canceled'
      }

      toast.success(t('toast.planClearedSuccess'))
      set({ importPlan: null, importResult: null })
      return true
    } catch (err) {
      toast.error(
        mapApiError(err, t('toast.cancelOrClearErrorFallback')).message,
      )
      return false
    } finally {
      set({ cancelingImportPlan: false })
    }
  },
}))
