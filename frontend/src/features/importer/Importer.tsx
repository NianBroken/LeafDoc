import {
  SettingsTable,
  SettingsTableCell,
} from '@/shared/components/SettingsTable'
import { useImporter } from './useImporter'

import { Button } from '@/shared/components/ui/button'

import i18next from '@/shared/lib/i18n'

import { FileUp, Loader2, PlayIcon, UploadIcon, XIcon } from 'lucide-react'

import { type ResultFilter } from '@/features/importer/importerState'

const t = (key: string, opts?: Record<string, unknown>) =>
  i18next.t(key, { ...opts, ns: 'importer' })

function getPlanActionLabel(action: 'create' | 'update' | 'skip'): string {
  switch (action) {
    case 'create':
      return t('planActions.create')
    case 'update':
      return t('planActions.update')
    case 'skip':
      return t('planActions.skip')
  }
}

function getPlanActionClass(action: 'create' | 'update' | 'skip'): string {
  switch (action) {
    case 'create':
      return 'settings__pill settings__pill-success'
    case 'update':
      return 'settings__pill settings__pill-warning'
    case 'skip':
      return 'settings__pill settings__pill-warning'
  }
}

function getResultActionLabel(
  action: 'created' | 'updated' | 'skipped' | 'conflicted',
  hasError: boolean,
): string {
  if (hasError) {
    return t('resultActions.needsAttention')
  }

  switch (action) {
    case 'created':
      return t('resultActions.created')
    case 'updated':
      return t('resultActions.updated')
    case 'skipped':
      return t('resultActions.skipped')
    case 'conflicted':
      return t('resultActions.conflicted')
  }
}

function getResultActionClass(
  action: 'created' | 'updated' | 'skipped' | 'conflicted',
  hasError: boolean,
): string {
  if (hasError || action === 'conflicted') {
    return 'settings__pill settings__pill-error'
  }

  switch (action) {
    case 'created':
      return 'settings__pill settings__pill-success'
    case 'updated':
      return 'settings__pill settings__pill-warning'
    case 'skipped':
      return 'settings__pill settings__pill-warning'
  }
}

export default function Importer() {
  const {
    formatDateTime,
    zipRef,
    zipFileName,
    setZipFileName,
    resultFilter,
    setResultFilter,
    resultSearch,
    setResultSearch,
    executeImportPlan,
    cancelImportPlan,
    importResult,
    creatingImportPlan,
    loadingImportPlan,
    executingImportPlan,
    cancelingImportPlan,
    importPlan,
    createImportPlanFromZip,
    closeImporter,
    startNewImport,
    importStatus,
    progressLabel,
    progressPercent,
    statusPillClass,
    statusLabel,
    clearButtonLabel,
    currentStep,
    stepItems,
    plannedCreateCount,
    plannedUpdateCount,
    plannedSkipCount,
    filteredResultItems,
    failedCount,
    createdCount,
    skippedCount,
    currentStepItem,
    showPackageStep,
    showPlanStep,
    showRunStep,
    showResultStep,
    showRunSection,
    nextActionText,
  } = useImporter()
  return (
    <>
      <div className="settings importer">
        <h1 className="settings__title">{t('title')}</h1>
        <div className="settings__section">
          <h2 className="settings__section-title">
            {t('beforeYouStart.title')}
          </h2>
          <p className="settings__section-description">
            {t('beforeYouStart.description')}
          </p>
          <div className="importer__callout">
            <div className="importer__callout-title">
              {t('beforeYouStart.whatHappensNext')}
            </div>
            <div className="importer__callout-body">
              <strong>
                {t('beforeYouStart.stepPrefix', { number: currentStep })}
                {currentStepItem ? `: ${currentStepItem.title}` : ''}
              </strong>
              <span>{nextActionText}</span>
            </div>
          </div>
          <div className="importer__info-grid">
            <div className="importer__info-card">
              <div className="importer__info-title">
                {t('beforeYouStart.safeReviewTitle')}
              </div>
              <div className="importer__info-text">
                {t('beforeYouStart.safeReviewText')}
              </div>
            </div>
            <div className="importer__info-card">
              <div className="importer__info-title">
                {t('beforeYouStart.linksAssetsTitle')}
              </div>
              <div className="importer__info-text">
                {t('beforeYouStart.linksAssetsText')}
              </div>
            </div>
            <div className="importer__info-card">
              <div className="importer__info-title">
                {t('beforeYouStart.alphaTitle')}
              </div>
              <div className="importer__info-text">
                {t('beforeYouStart.alphaText')}
              </div>
            </div>
          </div>
        </div>
        <div className="settings__section">
          <h2 className="settings__section-title">{t('flow.title')}</h2>
          <div className="importer__steps">
            {stepItems.map((step) => {
              const state =
                step.number < currentStep
                  ? 'done'
                  : step.number === currentStep
                    ? 'active'
                    : 'upcoming'

              return (
                <div
                  key={step.number}
                  className={`importer__step importer__step--${state}`}
                >
                  <div className="importer__step-number">{step.number}</div>
                  <div>
                    <div className="importer__step-title">{step.title}</div>
                    <div className="importer__step-description">
                      {step.description}
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
        {showPackageStep && (
          <div className="settings__section">
            <h2 className="settings__section-title">{t('package.title')}</h2>
            <p className="settings__section-description">
              {t('package.description')}
            </p>
            <div className="settings__preview">
              <span className="settings__preview-label">
                {t('package.currentZip')}
              </span>
              {zipFileName ? (
                <>
                  <span className="settings__preview-filename">
                    {zipFileName}
                  </span>
                </>
              ) : (
                <span className="settings__preview-placeholder">
                  {t('package.noZipSelected')}
                </span>
              )}
            </div>
            <div className="settings__field">
              <input
                type="file"
                ref={zipRef}
                accept=".zip"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0]
                  setZipFileName(file?.name ?? '')
                }}
              />
              <Button
                variant="outline"
                onClick={() => {
                  zipRef.current?.click()
                }}
              >
                <FileUp className="mr-2 h-4 w-4" />
                {t('package.selectZipFile')}
              </Button>
              <div className="settings__hint">
                {t('package.supportedInputHint')}
              </div>
            </div>
            <Button
              variant="default"
              className="settings__save-button"
              onClick={createImportPlanFromZip}
              disabled={
                !zipFileName || creatingImportPlan || executingImportPlan
              }
            >
              {creatingImportPlan || loadingImportPlan ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <UploadIcon className="mr-2 h-4 w-4" />
              )}
              {t('package.importFromZip')}
            </Button>
            <div className="settings__hint">{t('package.planOnlyHint')}</div>
          </div>
        )}
        {importPlan && showPlanStep && (
          <div className="settings__section">
            <h2 className="settings__section-title">{t('plan.title')}</h2>
            <p className="settings__section-description">
              {t('plan.description')}
            </p>
            <div className="importer__status-banner">
              <div className="importer__status-header">
                <div>
                  <div className="settings__preview-label">
                    {t('plan.currentStatus')}
                  </div>
                  <div className="importer__status-title">{statusLabel}</div>
                </div>
                <span className={statusPillClass}>{statusLabel}</span>
              </div>
              <div className="importer__status-meta">
                <span>{t('plan.planId', { id: importPlan.id })}</span>
                {progressLabel ? (
                  <span>{t('plan.progress', { progress: progressLabel })}</span>
                ) : null}
                {importPlan.total_items > 0 ? (
                  <span>
                    {t('plan.percentComplete', { percent: progressPercent })}
                  </span>
                ) : null}
              </div>
              {importPlan.total_items > 0 ? (
                <div className="importer__progress">
                  <div
                    className="importer__progress-bar"
                    style={{ width: `${progressPercent}%` }}
                  />
                </div>
              ) : null}
            </div>
            <div className="importer__summary-grid importer__summary-grid--three">
              <div className="importer__summary-card">
                <div className="importer__summary-label">
                  {t('plan.willCreate')}
                </div>
                <div className="importer__summary-value">
                  {plannedCreateCount}
                </div>
              </div>
              <div className="importer__summary-card">
                <div className="importer__summary-label">
                  {t('plan.willUpdate')}
                </div>
                <div className="importer__summary-value">
                  {plannedUpdateCount}
                </div>
              </div>
              <div className="importer__summary-card">
                <div className="importer__summary-label">
                  {t('plan.willSkip')}
                </div>
                <div className="importer__summary-value">
                  {plannedSkipCount}
                </div>
              </div>
            </div>
            {importStatus === 'running' && (
              <p className="settings__section-description importer__status-copy">
                {importPlan.cancel_requested
                  ? t('plan.runningCancelRequested')
                  : t('plan.runningInBackground')}
                {importPlan.current_item_source_path
                  ? t('plan.currentlyProcessing', {
                      path: importPlan.current_item_source_path,
                    })
                  : ''}
              </p>
            )}
            {importStatus === 'canceled' && (
              <p className="settings__section-description importer__status-copy">
                {t('plan.canceledNotice')}
              </p>
            )}
            {importStatus === 'failed' && importPlan.execution_error && (
              <p className="settings__section-description importer__status-copy text-error">
                {importPlan.execution_error}
              </p>
            )}

            <div className="settings__table-card">
              <div className="settings__table-scroll">
                <SettingsTable columns={2}>
                  <tbody>
                    <tr>
                      <th className="settings__table-header-cell">
                        {t('plan.idLabel')}
                      </th>
                      <SettingsTableCell>{importPlan.id}</SettingsTableCell>
                    </tr>
                    <tr>
                      <th className="settings__table-header-cell">
                        {t('plan.treeHashLabel')}
                      </th>
                      <SettingsTableCell>
                        {importPlan.tree_hash}
                      </SettingsTableCell>
                    </tr>
                    <tr>
                      <th className="settings__table-header-cell">
                        {t('plan.progressLabel')}
                      </th>
                      <SettingsTableCell>
                        {progressLabel ?? '0/0'}
                      </SettingsTableCell>
                    </tr>
                    {importPlan.started_at && (
                      <tr>
                        <th className="settings__table-header-cell">
                          {t('plan.startedAtLabel')}
                        </th>
                        <SettingsTableCell>
                          {formatDateTime(importPlan.started_at)}
                        </SettingsTableCell>
                      </tr>
                    )}
                    {importPlan.finished_at && (
                      <tr>
                        <th className="settings__table-header-cell">
                          {t('plan.finishedAtLabel')}
                        </th>
                        <SettingsTableCell>
                          {formatDateTime(importPlan.finished_at)}
                        </SettingsTableCell>
                      </tr>
                    )}
                  </tbody>
                </SettingsTable>
              </div>
            </div>
            <div className="content-actions content-actions--start">
              <Button
                variant="outline"
                onClick={() => {
                  void startNewImport()
                }}
                disabled={cancelingImportPlan}
              >
                {cancelingImportPlan ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <UploadIcon className="mr-2 h-4 w-4" />
                )}
                {t('plan.startNewImport')}
              </Button>
              <Button
                variant="destructive"
                onClick={() => {
                  void closeImporter()
                }}
                disabled={cancelingImportPlan}
              >
                {cancelingImportPlan ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <XIcon className="mr-2 h-4 w-4" />
                )}
                {t('plan.closeAndClear')}
              </Button>
            </div>
          </div>
        )}
        {importPlan && showPlanStep && importPlan.items.length > 0 && (
          <>
            <div className="settings__section">
              <h2 className="settings__section-title">
                {t('plannedItems.title', { count: importPlan.items.length })}
              </h2>
              <p className="settings__section-description">
                {t('plannedItems.description')}
              </p>
              <div className="settings__field">
                <div className="settings__table-card">
                  <div className="settings__table-scroll">
                    <SettingsTable columns={6}>
                      <thead className="settings__table-head">
                        <tr>
                          <th className="settings__table-header-cell">
                            {t('plannedItems.importFile')}
                          </th>
                          <th className="settings__table-header-cell">
                            {t('plannedItems.targetPage')}
                          </th>
                          <th className="settings__table-header-cell">
                            {t('plannedItems.titleColumn')}
                          </th>
                          <th className="settings__table-header-cell">
                            {t('plannedItems.type')}
                          </th>
                          <th className="settings__table-header-cell">
                            {t('plannedItems.plannedAction')}
                          </th>
                          <th className="settings__table-header-cell">
                            {t('plannedItems.notesForReview')}
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {importPlan.items.map((item) => (
                          <tr key={`${item.source_path}::${item.target_path}`}>
                            <SettingsTableCell className="settings__table-cell">
                              {item.source_path}
                            </SettingsTableCell>
                            <SettingsTableCell className="settings__table-cell">
                              {item.target_path}
                            </SettingsTableCell>
                            <SettingsTableCell className="settings__table-cell">
                              {item.title}
                            </SettingsTableCell>
                            <SettingsTableCell className="settings__table-cell">
                              <span className="settings__pill settings__pill-success">
                                {item.kind}
                              </span>
                            </SettingsTableCell>
                            <SettingsTableCell className="settings__table-cell">
                              <span className={getPlanActionClass(item.action)}>
                                {getPlanActionLabel(item.action)}
                              </span>
                            </SettingsTableCell>
                            <SettingsTableCell className="settings__table-cell importer__details-cell">
                              {item.notes && item.notes.length > 0 ? (
                                item.notes.join(', ')
                              ) : (
                                <span className="importer__muted-copy">
                                  {t('plannedItems.noSpecialNotes')}
                                </span>
                              )}
                            </SettingsTableCell>
                          </tr>
                        ))}
                      </tbody>
                    </SettingsTable>
                  </div>
                </div>
              </div>
            </div>
          </>
        )}
        {importPlan && showRunSection && (
          <div className="settings__section">
            <h2 className="settings__section-title">{t('run.title')}</h2>
            <p className="settings__section-description">
              {showPlanStep
                ? t('run.descriptionPlan')
                : t('run.descriptionRunning')}
            </p>
            {showRunStep && (
              <>
                <div className="importer__status-banner">
                  <div className="importer__status-header">
                    <div>
                      <div className="settings__preview-label">
                        {t('run.currentStatus')}
                      </div>
                      <div className="importer__status-title">
                        {statusLabel}
                      </div>
                    </div>
                    <span className={statusPillClass}>{statusLabel}</span>
                  </div>
                  <div className="importer__status-meta">
                    <span>{t('plan.planId', { id: importPlan.id })}</span>
                    {progressLabel ? (
                      <span>
                        {t('plan.progress', { progress: progressLabel })}
                      </span>
                    ) : null}
                    {importPlan.total_items > 0 ? (
                      <span>
                        {t('plan.percentComplete', {
                          percent: progressPercent,
                        })}
                      </span>
                    ) : null}
                  </div>
                  {importPlan.total_items > 0 ? (
                    <div className="importer__progress">
                      <div
                        className="importer__progress-bar"
                        style={{ width: `${progressPercent}%` }}
                      />
                    </div>
                  ) : null}
                </div>
                <p className="settings__section-description importer__status-copy">
                  {importPlan.cancel_requested
                    ? t('plan.runningCancelRequested')
                    : t('plan.runningInBackground')}
                  {importPlan.current_item_source_path
                    ? t('plan.currentlyProcessing', {
                        path: importPlan.current_item_source_path,
                      })
                    : ''}
                </p>
              </>
            )}
            <div className="content-actions content-actions--start">
              <Button
                variant="default"
                onClick={() => {
                  executeImportPlan()
                }}
                disabled={
                  importPlan.items.length === 0 ||
                  executingImportPlan ||
                  importStatus === 'completed' ||
                  importStatus === 'canceled' ||
                  creatingImportPlan ||
                  cancelingImportPlan
                }
              >
                {executingImportPlan ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <PlayIcon className="mr-2 h-4 w-4" />
                )}
                {importStatus === 'running'
                  ? t('run.importRunning')
                  : t('run.executeImportPlan')}
              </Button>
              <Button
                variant="destructive"
                onClick={() => {
                  cancelImportPlan()
                }}
                disabled={
                  cancelingImportPlan ||
                  (importStatus === 'running' && importPlan.cancel_requested) ||
                  creatingImportPlan
                }
              >
                {cancelingImportPlan ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <XIcon className="mr-2 h-4 w-4" />
                )}
                {clearButtonLabel}
              </Button>
            </div>
          </div>
        )}
        {importResult && showResultStep && (
          <div className="settings__section">
            <h2 className="settings__section-title">{t('result.title')}</h2>
            <p className="settings__section-description">
              {t('result.description')}
            </p>
            <div className="importer__callout">
              <div className="importer__callout-title">
                {t('result.wantAnotherTitle')}
              </div>
              <div className="importer__callout-body">
                <span>{t('result.wantAnotherBody')}</span>
              </div>
            </div>
            <div className="importer__summary-grid">
              <div className="importer__summary-card">
                <div className="importer__summary-label">
                  {t('result.created')}
                </div>
                <div className="importer__summary-value">{createdCount}</div>
              </div>
              <div className="importer__summary-card">
                <div className="importer__summary-label">
                  {t('result.skipped')}
                </div>
                <div className="importer__summary-value">{skippedCount}</div>
              </div>
              <div className="importer__summary-card">
                <div className="importer__summary-label">
                  {t('result.failed')}
                </div>
                <div className="importer__summary-value">{failedCount}</div>
              </div>
              <div className="importer__summary-card">
                <div className="importer__summary-label">
                  {t('result.treeUpdated')}
                </div>
                <div className="importer__summary-value">
                  {importResult.tree_hash === importResult.tree_hash_before
                    ? t('result.no')
                    : t('result.yes')}
                </div>
              </div>
            </div>

            <div className="settings__table-card">
              <div className="settings__table-scroll">
                <SettingsTable columns={2}>
                  <tbody>
                    <tr>
                      <th className="settings__table-header-cell">
                        {t('result.importedCount')}
                      </th>
                      <SettingsTableCell>
                        {importResult.imported_count}
                      </SettingsTableCell>
                    </tr>
                    <tr>
                      <th className="settings__table-header-cell">
                        {t('result.updatedCount')}
                      </th>
                      <SettingsTableCell>
                        {importResult.updated_count}
                      </SettingsTableCell>
                    </tr>
                    <tr>
                      <th className="settings__table-header-cell">
                        {t('result.skippedCount')}
                      </th>
                      <SettingsTableCell>
                        {importResult.skipped_count}
                      </SettingsTableCell>
                    </tr>
                  </tbody>
                </SettingsTable>
              </div>
            </div>
            <div className="content-actions content-actions--start">
              <Button
                variant="outline"
                onClick={() => {
                  void startNewImport()
                }}
                disabled={cancelingImportPlan}
              >
                {cancelingImportPlan ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <UploadIcon className="mr-2 h-4 w-4" />
                )}
                {t('plan.startNewImport')}
              </Button>
              <Button
                variant="destructive"
                onClick={() => {
                  void closeImporter()
                }}
                disabled={cancelingImportPlan}
              >
                {cancelingImportPlan ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <XIcon className="mr-2 h-4 w-4" />
                )}
                {t('plan.closeAndClear')}
              </Button>
            </div>
          </div>
        )}
        {importResult && showResultStep && importResult.items.length > 0 && (
          <div className="settings__section">
            <h2 className="settings__section-title">
              {t('resultItems.title', {
                filtered: filteredResultItems.length,
                total: importResult.items.length,
              })}
            </h2>
            <p className="settings__section-description">
              {t('resultItems.description')}
            </p>
            <div className="importer__result-toolbar">
              <div className="importer__filter-group">
                {(
                  ['all', 'created', 'skipped', 'failed'] as ResultFilter[]
                ).map((filter) => (
                  <button
                    key={filter}
                    type="button"
                    className={`importer__filter-chip ${
                      resultFilter === filter
                        ? 'importer__filter-chip--active'
                        : ''
                    }`}
                    onClick={() => {
                      setResultFilter(filter)
                    }}
                  >
                    {t(`resultItems.filter.${filter}`)}
                  </button>
                ))}
              </div>
              <input
                type="search"
                className="importer__search"
                placeholder={t('resultItems.filterPlaceholder')}
                value={resultSearch}
                onChange={(e) => {
                  setResultSearch(e.target.value)
                }}
              />
            </div>
            <div className="settings__table-card">
              <div className="settings__table-scroll">
                <SettingsTable columns={4}>
                  <thead className="settings__table-head">
                    <tr>
                      <th className="settings__table-header-cell">
                        {t('resultItems.importedFile')}
                      </th>
                      <th className="settings__table-header-cell">
                        {t('resultItems.resultingPage')}
                      </th>
                      <th className="settings__table-header-cell">
                        {t('resultItems.outcome')}
                      </th>
                      <th className="settings__table-header-cell">
                        {t('resultItems.details')}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredResultItems.map((item) => (
                      <tr key={`${item.source_path}::${item.target_path}`}>
                        <SettingsTableCell className="settings__table-cell">
                          {item.source_path}
                        </SettingsTableCell>
                        <SettingsTableCell className="settings__table-cell">
                          {item.target_path}
                        </SettingsTableCell>
                        <SettingsTableCell className="settings__table-cell">
                          <span
                            className={getResultActionClass(
                              item.action,
                              Boolean(item.error),
                            )}
                          >
                            {getResultActionLabel(
                              item.action,
                              Boolean(item.error),
                            )}
                          </span>
                        </SettingsTableCell>
                        <SettingsTableCell className="settings__table-cell importer__details-cell">
                          {item.error ? (
                            item.error
                          ) : item.action === 'created' ? (
                            t('resultItems.detailImported')
                          ) : item.action === 'updated' ? (
                            t('resultItems.detailUpdated')
                          ) : item.action === 'skipped' ? (
                            t('resultItems.detailSkipped')
                          ) : (
                            <span className="importer__muted-copy">
                              {t('resultItems.detailNone')}
                            </span>
                          )}
                        </SettingsTableCell>
                      </tr>
                    ))}
                    {filteredResultItems.length === 0 && (
                      <tr>
                        <SettingsTableCell
                          className="settings__table-body-message"
                          colSpan={4}
                        >
                          {t('resultItems.noItemsMatch')}
                        </SettingsTableCell>
                      </tr>
                    )}
                  </tbody>
                </SettingsTable>
              </div>
            </div>
          </div>
        )}
      </div>
    </>
  )
}
