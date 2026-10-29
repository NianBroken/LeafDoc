// Coordinates task state and actions independently of the settings view.

import { ApiLocalizedError, mapApiError } from '@/shared/lib/api/errors'

import { useDateTimeFormat } from '@/shared/lib/useDateTimeFormat'

import { useEffect, useRef, useState } from 'react'

import { useTranslation } from 'react-i18next'

import { toast } from 'sonner'

import { useSnapshotStore } from '@/features/snapshot/state/snapshot'

import type { SnapshotSettings as SnapshotSettingsValue } from '@/shared/lib/api/snapshot'

import { useRestoreStore } from '@/features/restore/state/restore'

import { useSetTitle } from '@/features/viewer/setTitle'
import { useSessionStore } from '@/shared/state/session'
import { BASE_PATH } from '@/shared/lib/config'

// Restored account data invalidates every previous browser session.
function returnToLogin() {
  useSessionStore.getState().clear()
  window.location.replace(`${BASE_PATH}/login`)
}

export function useSnapshotSettings() {
  const { t } = useTranslation('snapshot')
  const { formatDateTime } = useDateTimeFormat()
  const { t: tRestore } = useTranslation('restore')
  const {
    enabled,
    retentionCount,
    settings,
    isRunning,
    lastSnapshotAt,
    lastError,
    lastPruneError,
    snapshots,
    isLoading,
    isListLoading,
    statusError,
    loadStatus,
    loadList,
    triggerNow,
    saveSettings,
    remove,
  } = useSnapshotStore()
  const {
    isLoading: isRestoring,
    phase: restorePhase,
    isResyncPhase,
    needsIntervention,
    versionWarning,
    trigger: triggerRestore,
    triggerUpload,
    selfRestart,
  } = useRestoreStore()

  const [isTriggering, setIsTriggering] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [restoringId, setRestoringId] = useState<string | null>(null)
  const [isSelfRestarting, setIsSelfRestarting] = useState(false)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [isRestoringUpload, setIsRestoringUpload] = useState(false)
  const [settingsDraft, setSettingsDraft] =
    useState<SnapshotSettingsValue | null>(null)
  const [retentionInput, setRetentionInput] = useState('')
  const [settingsSaving, setSettingsSaving] = useState(false)
  const [settingsError, setSettingsError] = useState('')
  const uploadInputRef = useRef<HTMLInputElement>(null)

  useSetTitle({ title: t('pageTitle') })

  useEffect(() => {
    loadStatus()
    loadList()
  }, [loadStatus, loadList])

  useEffect(() => {
    if (settings) {
      setSettingsDraft(settings)
      setRetentionInput(String(settings.retentionCount))
    }
  }, [settings])

  const handleSettingsSave = async () => {
    if (!settingsDraft || !/^(0|[1-9]\d*)$/.test(retentionInput)) {
      setSettingsError(t('settings.invalidRetention'))
      return
    }
    const count = Number(retentionInput)
    if (!Number.isSafeInteger(count) || count > 100000) {
      setSettingsError(t('settings.invalidRetention'))
      return
    }
    const value = {
      ...settingsDraft,
      retentionCount: count,
      runOnStartup:
        settingsDraft.enabled && settingsDraft.mode === 'interval'
          ? (settings?.runOnStartup ?? false)
          : false,
    }
    setSettingsSaving(true)
    setSettingsError('')
    try {
      await saveSettings(value)
      toast.success(t('settings.savedToast'))
    } catch (error) {
      setSettingsError(
        error instanceof Error ? error.message : t('settings.saveError'),
      )
    } finally {
      setSettingsSaving(false)
    }
  }

  const handleTrigger = async () => {
    setIsTriggering(true)
    try {
      await triggerNow()
      toast.success(t('toast.backupTriggered'))
    } catch {
      toast.error(t('toast.backupTriggerFailed'))
    } finally {
      setIsTriggering(false)
    }
  }

  const handleDelete = async (id: string) => {
    setDeletingId(id)
    try {
      await remove(id)
      toast.success(t('toast.deleteSuccess'))
    } catch {
      toast.error(t('toast.deleteFailed'))
    } finally {
      setDeletingId(null)
    }
  }

  const handleRestore = async (id: string) => {
    setRestoringId(id)
    try {
      await triggerRestore(id)
      if (useRestoreStore.getState().needsIntervention) {
        return
      }
      if (useRestoreStore.getState().resyncConfirmed) {
        toast.success(tRestore('toast.restoreSucceeded'))
      } else {
        toast.warning(tRestore('toast.restoreSucceededResyncUnconfirmed'))
      }
      returnToLogin()
    } catch (err) {
      if (
        err instanceof ApiLocalizedError &&
        err.code === 'restore_already_running'
      ) {
        toast.info(tRestore('toast.restoreAlreadyRunning'))
      } else {
        toast.error(mapApiError(err, tRestore('toast.restoreFailed')).message)
      }
    } finally {
      setRestoringId(null)
    }
  }

  const handleRestoreFromUpload = async () => {
    if (!uploadFile) return
    setIsRestoringUpload(true)
    try {
      await triggerUpload(uploadFile)
      if (useRestoreStore.getState().needsIntervention) {
        return
      }
      if (useRestoreStore.getState().resyncConfirmed) {
        toast.success(tRestore('toast.restoreSucceeded'))
      } else {
        toast.warning(tRestore('toast.restoreSucceededResyncUnconfirmed'))
      }
      returnToLogin()
    } catch (err) {
      if (
        err instanceof ApiLocalizedError &&
        err.code === 'restore_already_running'
      ) {
        toast.info(tRestore('toast.restoreAlreadyRunning'))
      } else {
        toast.error(mapApiError(err, tRestore('toast.restoreFailed')).message)
      }
    } finally {
      setIsRestoringUpload(false)
    }
  }

  const handleSelfRestart = async () => {
    setIsSelfRestarting(true)
    await selfRestart()
    // The connection drops as the process replaces itself; give it a moment
    // to come back up before reloading into the fresh instance.
    window.setTimeout(() => window.location.reload(), 5000)
  }

  const restorePhaseLabel = restorePhase
    ? tRestore(`progress.${restorePhase}`, { defaultValue: restorePhase })
    : '…'

  return {
    t,
    formatDateTime,
    tRestore,
    enabled,
    retentionCount,
    isRunning,
    lastSnapshotAt,
    lastError,
    lastPruneError,
    snapshots,
    isLoading,
    isListLoading,
    statusError,
    isRestoring,
    isResyncPhase,
    needsIntervention,
    versionWarning,
    isTriggering,
    deletingId,
    restoringId,
    isSelfRestarting,
    uploadFile,
    setUploadFile,
    isRestoringUpload,
    settingsDraft,
    setSettingsDraft,
    retentionInput,
    setRetentionInput,
    settingsSaving,
    settingsError,
    uploadInputRef,
    handleSettingsSave,
    handleTrigger,
    handleDelete,
    handleRestore,
    handleRestoreFromUpload,
    handleSelfRestart,
    restorePhaseLabel,
  }
}
