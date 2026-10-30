import { getIntervalOptions, getTimeZoneOptions } from './scheduleOptions'
import {
  downwardSelectContentProps,
  revealDownwardSelect,
} from '@/shared/lib/selectPlacement'
import { Checkbox } from '@/shared/components/ui/checkbox'
import {
  SettingsTable,
  SettingsTableCell,
  SettingsTableHeader,
} from '@/shared/components/SettingsTable'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/components/ui/select'
import { useSnapshotSettings } from './useSnapshotSettings'

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
} from '@/shared/components/ui/alert-dialog'

import { Button } from '@/shared/components/ui/button'

import { Input } from '@/shared/components/ui/input'

import { Label } from '@/shared/components/ui/label'

import { formatBytes } from '@/shared/lib/config'

import { snapshotDownloadUrl } from '@/shared/lib/api/snapshot'

import {
  AlertTriangle,
  FileUp,
  HardDriveDownload,
  History,
  Loader2,
  SaveIcon,
} from 'lucide-react'

export default function SnapshotSettings() {
  const {
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
  } = useSnapshotSettings()
  const intervalChoices = getIntervalOptions(settingsDraft?.interval ?? '')
  const timeZoneChoices = getTimeZoneOptions(settingsDraft?.timeZone ?? '')
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
        <div
          className="settings__section"
          data-testid="snapshot-schedule-settings"
        >
          <h2 className="settings__section-title">{t('settings.title')}</h2>
          <p className="settings__section-description">
            {t('settings.description')}
          </p>
          <div className="settings__field">
            <label className="settings__schedule-toggle">
              <Checkbox
                checked={settingsDraft.enabled}
                onCheckedChange={(checked) =>
                  setSettingsDraft({
                    ...settingsDraft,
                    enabled: checked === true,
                    runOnStartup: false,
                  })
                }
                disabled={settingsSaving}
                className="mt-0.5"
                data-testid="snapshot-schedule-enabled"
              />
              <span>
                <span className="text-interface-text block text-sm font-medium">
                  {t('settings.enabledLabel')}
                </span>
                <span className="settings__hint block">
                  {t('settings.enabledDescription')}
                </span>
              </span>
            </label>
          </div>
          <div className="settings__schedule-grid">
            <div className="settings__field">
              <Label htmlFor="snapshot-schedule-mode">
                {t('settings.modeLabel')}
              </Label>
              <Select
                value={settingsDraft.mode}
                onOpenChange={(open) =>
                  revealDownwardSelect('snapshot-schedule-mode', open)
                }
                disabled={!settingsDraft.enabled || settingsSaving}
                onValueChange={(value) =>
                  setSettingsDraft({
                    ...settingsDraft,
                    mode: value as 'interval' | 'daily',
                    runOnStartup: false,
                  })
                }
              >
                <SelectTrigger
                  id="snapshot-schedule-mode"
                  aria-label={t('settings.modeLabel')}
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent {...downwardSelectContentProps}>
                  <SelectItem value="daily">
                    {t('settings.dailyMode')}
                  </SelectItem>
                  <SelectItem value="interval">
                    {t('settings.intervalMode')}
                  </SelectItem>
                </SelectContent>
              </Select>
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
                <Select
                  value={intervalChoices.value}
                  onOpenChange={(open) =>
                    revealDownwardSelect('snapshot-schedule-interval', open)
                  }
                  disabled={!settingsDraft.enabled || settingsSaving}
                  onValueChange={(interval) =>
                    setSettingsDraft({ ...settingsDraft, interval })
                  }
                >
                  <SelectTrigger
                    id="snapshot-schedule-interval"
                    aria-label={t('settings.intervalLabel')}
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent {...downwardSelectContentProps}>
                    {intervalChoices.options.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.unit
                          ? t(`settings.intervalUnits.${option.unit}`, {
                              count: option.count,
                            })
                          : t('settings.currentInterval', {
                              value: option.value,
                            })}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
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
                <Select
                  value={settingsDraft.timeZone}
                  onOpenChange={(open) =>
                    revealDownwardSelect('snapshot-schedule-timezone', open)
                  }
                  disabled={!settingsDraft.enabled || settingsSaving}
                  onValueChange={(timeZone) =>
                    setSettingsDraft({ ...settingsDraft, timeZone })
                  }
                >
                  <SelectTrigger
                    id="snapshot-schedule-timezone"
                    aria-label={t('settings.timeZoneLabel')}
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent {...downwardSelectContentProps}>
                    {timeZoneChoices.map((timeZone) => (
                      <SelectItem key={timeZone} value={timeZone}>
                        {timeZone}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
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
            <p role="alert" className="settings__notice text-error">
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
              <div className="settings__table-card">
                <div className="settings__table-scroll">
                  <SettingsTable columns={4} className="snapshot__table">
                    <thead className="settings__table-head">
                      <tr>
                        <SettingsTableHeader>
                          {t('columnName')}
                        </SettingsTableHeader>
                        <SettingsTableHeader>
                          {t('columnDate')}
                        </SettingsTableHeader>
                        <SettingsTableHeader>
                          {t('columnSize')}
                        </SettingsTableHeader>
                        <SettingsTableHeader>
                          {t('columnActions')}
                        </SettingsTableHeader>
                      </tr>
                    </thead>
                    <tbody>
                      {snapshots.map((snap) => (
                        <tr key={snap.id} className="settings__table-row">
                          <SettingsTableCell>{snap.id}</SettingsTableCell>
                          <SettingsTableCell>
                            {formatDateTime(snap.createdAt) || snap.id}
                          </SettingsTableCell>
                          <SettingsTableCell>
                            {formatBytes(snap.sizeBytes)}
                          </SettingsTableCell>
                          <SettingsTableCell content="controls">
                            <div className="settings__table-actions">
                              <a
                                className="settings__text-action"
                                href={snapshotDownloadUrl(snap.id)}
                                download
                              >
                                {t('download')}
                              </a>
                              <AlertDialog>
                                <AlertDialogTrigger asChild>
                                  <button
                                    type="button"
                                    className="settings__text-action"
                                    disabled={
                                      isRestoring ||
                                      restoringId === snap.id ||
                                      needsIntervention
                                    }
                                  >
                                    {restoringId === snap.id ? (
                                      <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                                    ) : null}
                                    {tRestore('restoreButton')}
                                  </button>
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
                                  <button
                                    type="button"
                                    disabled={deletingId === snap.id}
                                    className="settings__text-action text-error"
                                  >
                                    {deletingId === snap.id ? (
                                      <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />
                                    ) : null}
                                    {t('delete')}
                                  </button>
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
                          </SettingsTableCell>
                        </tr>
                      ))}
                    </tbody>
                  </SettingsTable>
                </div>
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
