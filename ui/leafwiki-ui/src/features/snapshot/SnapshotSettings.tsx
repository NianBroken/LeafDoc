import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { formatBytes } from '@/lib/config'
import { snapshotDownloadUrl } from '@/lib/api/snapshot'
import { ApiLocalizedError, mapApiError } from '@/lib/api/errors'
import {
  AlertTriangle,
  Download,
  FileUp,
  HardDriveDownload,
  History,
  Loader2,
  SaveIcon,
  Trash2,
} from 'lucide-react'
import { useDateTimeFormat } from '@/lib/useDateTimeFormat'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useSnapshotStore } from '@/stores/snapshot'
import type { SnapshotSettings as SnapshotSettingsValue } from '@/lib/api/snapshot'
import { useRestoreStore } from '@/stores/restore'
import { useSetTitle } from '../viewer/setTitle'

export default function SnapshotSettings() {
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
  const [settingsDraft, setSettingsDraft] = useState<SnapshotSettingsValue | null>(null)
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
          ? settings?.runOnStartup ?? false
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
      window.location.reload()
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
      window.location.reload()
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

  return (
    <div className="settings">
      <h1 className="settings__title">{t('pageTitle')}</h1>

      {statusError && (
        <div className="settings__section">
          <p className="text-error text-sm">{statusError}</p>
        </div>
      )}

      {needsIntervention && (
        <div className="settings__section border-error/20 bg-error/5">
          <h2 className="settings__section-title text-error flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            {tRestore('needsInterventionTitle')}
          </h2>
          <p className="settings__section-description">
            {tRestore('needsInterventionDescription')}
          </p>
          <div className="settings__actions">
            <Button
              variant="destructive"
              onClick={handleSelfRestart}
              disabled={isSelfRestarting}
            >
              {isSelfRestarting ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <AlertTriangle className="mr-2 h-4 w-4" />
              )}
              {isSelfRestarting
                ? tRestore('selfRestarting')
                : tRestore('selfRestartButton')}
            </Button>
          </div>
        </div>
      )}

      {isRestoring && (
        <div className="settings__section">
          <div className="importer__status-banner">
            <div className="importer__status-header">
              <div>
                <div className="settings__preview-label">
                  {tRestore('progressLabel')}
                </div>
                <div className="importer__status-title">
                  {restorePhaseLabel}
                </div>
                {isResyncPhase && (
                  <div className="text-muted text-xs">
                    {tRestore('resyncTailLabel')}
                  </div>
                )}
              </div>
            </div>
          </div>
          {versionWarning && (
            <p className="settings__hint text-warning mt-2">
              <strong>{tRestore('versionWarningLabel')}:</strong>{' '}
              {versionWarning}
            </p>
          )}
        </div>
      )}

      <div className="settings__section">
        <h2 className="settings__section-title">{t('sectionTitle')}</h2>
        <p className="settings__section-description">
          {t('sectionDescription')}
        </p>

        <div className="settings__preview">
          <span className="settings__preview-label">{t('statusLabel')}</span>
          {isLoading ? (
            <Loader2 className="text-muted h-4 w-4 animate-spin" />
          ) : enabled ? (
            <span className="settings__pill settings__pill-success text-success font-medium">
              {t('statusEnabled')}
            </span>
          ) : (
            <span className="settings__role-pill settings__role-pill--default">
              {t('statusDisabled')}
            </span>
          )}
        </div>

        {(isLoading || enabled) && (
          <>
            <div className="settings__preview">
              <span className="settings__preview-label">
                {t('retentionLabel')}
              </span>
              <span className="text-interface-text text-sm">
                {retentionCount > 0
                  ? t('retentionValue', { count: retentionCount })
                  : t('retentionUnlimited')}
              </span>
            </div>

            <div className="settings__preview">
              <span className="settings__preview-label">
                {t('lastBackupLabel')}
              </span>
              <span className="text-interface-text text-sm">
                {isLoading ? (
                  <span className="text-muted flex items-center gap-2">
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    {t('loading')}
                  </span>
                ) : isRunning ? (
                  <span className="text-muted flex items-center gap-2">
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    {t('running')}
                  </span>
                ) : (
                  formatDateTime(lastSnapshotAt ?? undefined) || t('never')
                )}
              </span>
            </div>

            {!isLoading && lastError && (
              <div className="settings__preview border-error/20 bg-error/5">
                <span className="settings__preview-label">
                  {t('lastErrorLabel')}
                </span>
                <span className="text-error text-sm">{lastError}</span>
              </div>
            )}

            {!isLoading && !lastError && lastPruneError && (
              <div className="settings__preview border-warning/20 bg-warning/5">
                <span className="settings__preview-label">
                  {t('lastPruneErrorLabel')}
                </span>
                <span className="text-warning text-sm">{lastPruneError}</span>
              </div>
            )}
          </>
        )}

        {!isLoading && !enabled && (
          <p className="settings__hint">{t('hintDisabled')}</p>
        )}
      </div>

      {!isLoading && enabled && settingsDraft && (
        <div className="settings__section" data-testid="snapshot-schedule-settings">
          <h2 className="settings__section-title">{t('settings.title')}</h2>
          <p className="settings__section-description">
            {t('settings.description')}
          </p>
          <div className="settings__field">
            <label className="flex items-start gap-3">
              <input
                type="checkbox"
                checked={settingsDraft.enabled}
                onChange={(event) =>
                  setSettingsDraft({
                    ...settingsDraft,
                    enabled: event.target.checked,
                    runOnStartup: false,
                  })
                }
                className="mt-1 accent-brand"
                data-testid="snapshot-schedule-enabled"
              />
              <span>
                <span className="text-interface-text block text-sm font-medium">
                  {t('settings.enabledLabel')}
                </span>
                <span className="text-muted-foreground text-sm">
                  {t('settings.enabledDescription')}
                </span>
              </span>
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="settings__field">
              <Label htmlFor="snapshot-schedule-mode">
                {t('settings.modeLabel')}
              </Label>
              <select
                id="snapshot-schedule-mode"
                className="border-input bg-background h-10 w-full rounded-md border px-3 text-sm"
                value={settingsDraft.mode}
                disabled={!settingsDraft.enabled || settingsSaving}
                onChange={(event) =>
                  setSettingsDraft({
                    ...settingsDraft,
                    mode: event.target.value as 'interval' | 'daily',
                    runOnStartup: false,
                  })
                }
              >
                <option value="daily">{t('settings.dailyMode')}</option>
                <option value="interval">{t('settings.intervalMode')}</option>
              </select>
            </div>
            {settingsDraft.mode === 'daily' ? (
              <div className="settings__field">
                <Label htmlFor="snapshot-schedule-time">
                  {t('settings.timeLabel')}
                </Label>
                <Input
                  id="snapshot-schedule-time"
                  type="time"
                  value={`${String(settingsDraft.hour).padStart(2, '0')}:${String(settingsDraft.minute).padStart(2, '0')}`}
                  disabled={!settingsDraft.enabled || settingsSaving}
                  onChange={(event) => {
                    const [hour, minute] = event.target.value
                      .split(':')
                      .map(Number)
                    setSettingsDraft({ ...settingsDraft, hour, minute })
                  }}
                />
              </div>
            ) : (
              <div className="settings__field">
                <Label htmlFor="snapshot-schedule-interval">
                  {t('settings.intervalLabel')}
                </Label>
                <Input
                  id="snapshot-schedule-interval"
                  value={settingsDraft.interval}
                  placeholder="24h"
                  disabled={!settingsDraft.enabled || settingsSaving}
                  onChange={(event) =>
                    setSettingsDraft({
                      ...settingsDraft,
                      interval: event.target.value,
                    })
                  }
                />
                <p className="settings__hint">
                  {t('settings.intervalDescription')}
                </p>
              </div>
            )}
            {settingsDraft.mode === 'daily' && (
              <div className="settings__field sm:col-span-2">
                <Label htmlFor="snapshot-schedule-timezone">
                  {t('settings.timeZoneLabel')}
                </Label>
                <Input
                  id="snapshot-schedule-timezone"
                  value={settingsDraft.timeZone}
                  placeholder="Asia/Shanghai"
                  disabled={!settingsDraft.enabled || settingsSaving}
                  onChange={(event) =>
                    setSettingsDraft({
                      ...settingsDraft,
                      timeZone: event.target.value,
                    })
                  }
                />
                <p className="settings__hint">
                  {t('settings.timeZoneDescription')}
                </p>
              </div>
            )}
            <div className="settings__field">
              <Label htmlFor="snapshot-retention-count">
                {t('settings.retentionLabel')}
              </Label>
              <Input
                id="snapshot-retention-count"
                type="number"
                min={0}
                max={100000}
                step={1}
                value={retentionInput}
                disabled={settingsSaving}
                aria-invalid={Boolean(settingsError)}
                onChange={(event) => setRetentionInput(event.target.value)}
                data-testid="snapshot-retention-count"
              />
              <p className="settings__hint">
                {t('settings.retentionDescription')}
              </p>
            </div>
          </div>
          {settingsError && (
            <p role="alert" className="text-error text-sm">
              {settingsError}
            </p>
          )}
          <div className="settings__actions">
            <Button
              onClick={() => void handleSettingsSave()}
              disabled={settingsSaving}
              data-testid="snapshot-settings-save"
            >
              {settingsSaving ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <SaveIcon className="mr-2 h-4 w-4" />
              )}
              {t('settings.saveButton')}
            </Button>
          </div>
        </div>
      )}

      {!isLoading && enabled && (
        <>
          <div className="settings__section">
            <h2 className="settings__section-title">
              {t('manualSectionTitle')}
            </h2>
            <p className="settings__section-description">
              {t('manualSectionDescription')}
            </p>
            <div className="settings__actions">
              <Button onClick={handleTrigger} disabled={isTriggering}>
                {isTriggering ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <HardDriveDownload className="mr-2 h-4 w-4" />
                )}
                {isTriggering ? t('creating') : t('createNow')}
              </Button>
            </div>
          </div>

          <div className="settings__section">
            <h2 className="settings__section-title">{t('listSectionTitle')}</h2>
            <p className="settings__section-description">
              {t('listSectionDescription')}
            </p>

            {isListLoading ? (
              <Loader2 className="text-muted h-4 w-4 animate-spin" />
            ) : snapshots.length === 0 ? (
              <p className="settings__hint">{t('listEmpty')}</p>
            ) : (
              <div className="flex flex-col gap-2">
                {snapshots.map((snap) => (
                  <div key={snap.id} className="settings__preview">
                    <div className="flex flex-col">
                      <span className="text-interface-text text-sm font-medium">
                        {formatDateTime(snap.createdAt) || snap.id}
                      </span>
                      <span className="text-muted text-xs">
                        {formatBytes(snap.sizeBytes)}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="outline" size="sm" asChild>
                        <a href={snapshotDownloadUrl(snap.id)} download>
                          <Download className="mr-2 h-3.5 w-3.5" />
                          {t('download')}
                        </a>
                      </Button>
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={
                              isRestoring ||
                              restoringId === snap.id ||
                              needsIntervention
                            }
                          >
                            {restoringId === snap.id ? (
                              <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                            ) : (
                              <History className="mr-2 h-3.5 w-3.5" />
                            )}
                            {tRestore('restoreButton')}
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>
                              {tRestore('restoreConfirmTitle')}
                            </AlertDialogTitle>
                            <AlertDialogDescription>
                              {tRestore('restoreConfirmDescription')}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>
                              {tRestore('restoreConfirmCancel')}
                            </AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => handleRestore(snap.id)}
                            >
                              {tRestore('restoreConfirmAction')}
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={deletingId === snap.id}
                            className="text-error hover:bg-error/10"
                          >
                            {deletingId === snap.id ? (
                              <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                            ) : (
                              <Trash2 className="mr-2 h-3.5 w-3.5" />
                            )}
                            {t('delete')}
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>
                              {t('deleteConfirmTitle')}
                            </AlertDialogTitle>
                            <AlertDialogDescription>
                              {t('deleteConfirmDescription')}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>
                              {t('deleteConfirmCancel')}
                            </AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => handleDelete(snap.id)}
                            >
                              {t('deleteConfirmAction')}
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="settings__section">
            <h2 className="settings__section-title">
              {tRestore('uploadSectionTitle')}
            </h2>
            <p className="settings__section-description">
              {tRestore('uploadSectionDescription')}
            </p>

            <div className="settings__preview">
              <span className="settings__preview-label">
                {tRestore('currentFileLabel')}
              </span>
              {uploadFile ? (
                <span className="settings__preview-filename">
                  {uploadFile.name}
                </span>
              ) : (
                <span className="settings__preview-placeholder">
                  {tRestore('noFileSelected')}
                </span>
              )}
            </div>

            <div className="settings__actions">
              <input
                type="file"
                ref={uploadInputRef}
                accept=".zip"
                className="hidden"
                onChange={(e) => {
                  setUploadFile(e.target.files?.[0] ?? null)
                }}
              />
              <Button
                variant="outline"
                onClick={() => uploadInputRef.current?.click()}
              >
                <FileUp className="mr-2 h-4 w-4" />
                {tRestore('uploadSelectButton')}
              </Button>

              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    disabled={
                      !uploadFile ||
                      isRestoring ||
                      isRestoringUpload ||
                      needsIntervention
                    }
                  >
                    {isRestoringUpload ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <History className="mr-2 h-4 w-4" />
                    )}
                    {tRestore('restoreButton')}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {tRestore('uploadConfirmTitle')}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {tRestore('uploadConfirmDescription')}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>
                      {tRestore('uploadConfirmCancel')}
                    </AlertDialogCancel>
                    <AlertDialogAction onClick={handleRestoreFromUpload}>
                      {tRestore('uploadConfirmAction')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
