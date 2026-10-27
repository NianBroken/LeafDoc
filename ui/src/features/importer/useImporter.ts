import i18next from '@/shared/lib/i18n'
const t = (key: string, opts?: Record<string, unknown>) =>
  i18next.t(key, { ...opts, ns: 'importer' })
// Coordinates task state and actions independently of the settings view.

import { useDateTimeFormat } from '@/shared/lib/useDateTimeFormat'

import { useImportStore } from '@/features/importer/state/import'

import { useCallback, useEffect, useRef, useState } from 'react'

import { useNavigate } from 'react-router'

import { useSetTitle } from '@/features/viewer/setTitle'

import {
  filterResultItems,
  getClearButtonLabelKey,
  getImportProgress,
  getImportStep,
  getNextActionKey,
  getStatusLabelKey,
  getStatusPillClass,
  type ResultFilter,
} from '@/features/importer/importerState'

export function useImporter() {
  useSetTitle({ title: t('title') })
  const { formatDateTime } = useDateTimeFormat()
  const navigate = useNavigate()
  const zipRef = useRef<HTMLInputElement>(null)
  const [zipFileName, setZipFileName] = useState('')
  const [resultFilter, setResultFilter] = useState<ResultFilter>('all')
  const [resultSearch, setResultSearch] = useState('')

  const createImportPlan = useImportStore((store) => store.createImportPlan)
  const executeImportPlan = useImportStore((store) => store.executeImportPlan)
  const cancelImportPlan = useImportStore((store) => store.cancelImportPlan)
  const importResult = useImportStore((store) => store.importResult)
  const creatingImportPlan = useImportStore((store) => store.creatingImportPlan)
  const loadingImportPlan = useImportStore((store) => store.loadingImportPlan)
  const executingImportPlan = useImportStore(
    (store) => store.executingImportPlan,
  )
  const cancelingImportPlan = useImportStore(
    (store) => store.cancelingImportPlan,
  )

  const importPlan = useImportStore((store) => store.importPlan)
  const loadImportPlan = useImportStore((store) => store.loadImportPlan)

  useEffect(() => {
    void loadImportPlan()
  }, [loadImportPlan])

  const createImportPlanFromZip = useCallback(() => {
    const zipFile = zipRef.current?.files?.[0]
    if (!zipFile) {
      return
    }
    void createImportPlan(zipFile)
  }, [createImportPlan])

  const closeImporter = useCallback(async () => {
    const cleared = importPlan ? await cancelImportPlan() : true
    if (!cleared) {
      return
    }
    navigate('/')
  }, [cancelImportPlan, importPlan, navigate])

  const startNewImport = useCallback(async () => {
    const cleared = importPlan ? await cancelImportPlan() : true
    if (!cleared) {
      return
    }
    if (zipRef.current) {
      zipRef.current.value = ''
    }
    setZipFileName('')
    setResultFilter('all')
    setResultSearch('')
  }, [cancelImportPlan, importPlan])

  const importStatus = importPlan?.execution_status ?? null
  const { label: progressLabel, percent: progressPercent } =
    getImportProgress(importPlan)
  const statusPillClass = getStatusPillClass(importStatus)
  const statusLabel = t(getStatusLabelKey(importStatus))
  const clearButtonLabel = t(getClearButtonLabelKey(importPlan))
  const currentStep = getImportStep(importPlan, importResult)
  const stepItems = [
    {
      number: 1,
      title: t('steps.selectZip.title'),
      description: t('steps.selectZip.description'),
    },
    {
      number: 2,
      title: t('steps.reviewPlan.title'),
      description: t('steps.reviewPlan.description'),
    },
    {
      number: 3,
      title: t('steps.runImport.title'),
      description: t('steps.runImport.description'),
    },
    {
      number: 4,
      title: t('steps.reviewResult.title'),
      description: t('steps.reviewResult.description'),
    },
  ]
  const planItems = importPlan?.items ?? []
  const resultItems = importResult?.items ?? []
  const plannedCreateCount = planItems.filter(
    (item) => item.action === 'create',
  ).length
  const plannedUpdateCount = planItems.filter(
    (item) => item.action === 'update',
  ).length
  const plannedSkipCount = planItems.filter(
    (item) => item.action === 'skip',
  ).length
  const filteredResultItems = filterResultItems(
    resultItems,
    resultFilter,
    resultSearch,
  )
  const failedCount = resultItems.filter((item) => Boolean(item.error)).length
  const createdCount = resultItems.filter(
    (item) => item.action === 'created',
  ).length
  const skippedCount = resultItems.filter(
    (item) => item.action === 'skipped',
  ).length
  const currentStepItem = stepItems.find((step) => step.number === currentStep)
  const showPackageStep = currentStep === 1
  const showPlanStep = currentStep === 2
  const showRunStep = currentStep === 3
  const showResultStep = currentStep === 4
  const showRunSection = currentStep === 2 || currentStep === 3
  const nextActionText = t(getNextActionKey(importPlan))

  return {
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
  }
}
