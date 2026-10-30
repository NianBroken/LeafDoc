import Page404 from '@/shared/components/Page404'
import { buildViewUrl } from '@/shared/lib/routePath'
import {
  createNavigationVisitState,
  getNavigationVisitKey,
} from '@/shared/lib/navigationVisit'
import { useScrollRestoration } from '@/shared/lib/useScrollRestoration'
import { useTreeStore } from '@/features/tree/state/tree'
import { useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useExitModeButton } from '@/features/toolbar/useExitModeButton'
import { getWikiTargetRoutePath, toWikiLookupPath } from '@/shared/lib/wikiPath'
import { useLocation, useNavigate } from 'react-router'
import { useProgressbarStore } from '@/features/progressbar/progressbarStore'
import { useSetPageTitle } from '@/features/viewer/useSetPageTitle'
import { useViewerStore } from '@/features/viewer/viewer'
import { PageHistoryContent } from '@/features/history/PageHistoryContent'
import { usePageHistory } from '@/features/history/pageHistory'

export default function PageHistoryPage() {
  const { t } = useTranslation('page')
  const location = useLocation()
  const { pathname } = location
  const navigate = useNavigate()
  const openNode = useTreeStore((state) => state.openNode)
  const loading = useProgressbarStore((s) => s.loading)
  const error = useViewerStore((s) => s.error)
  const notFound = useViewerStore((s) => s.notFound)
  const page = useViewerStore((s) => s.page)
  const loadPageData = useViewerStore((s) => s.loadPageData)

  usePageHistory(page?.id ?? null)

  const closeHistory = useCallback(() => {
    navigate(buildViewUrl(page?.path || pathname), {
      state: createNavigationVisitState(),
    })
  }, [navigate, page?.path, pathname])

  useScrollRestoration(getNavigationVisitKey(location), loading)
  useSetPageTitle({ page })

  useEffect(() => {
    const path = toWikiLookupPath(buildViewUrl(pathname))
    void loadPageData?.(path)
  }, [pathname, loadPageData])

  useEffect(() => {
    if (!page?.id) return
    openNode(page.id)
  }, [openNode, page?.id])

  useExitModeButton({
    id: 'close-history',
    labelKey: 'historyPage.backToPage',
    ns: 'page',
    hotkeyId: 'history.page.close',
    onExit: closeHistory,
  })

  const renderError = () => {
    if (!loading && notFound) {
      return <Page404 targetPath={getWikiTargetRoutePath(pathname)} />
    }
    if (!loading && error) {
      return (
        <p className="page-viewer__error">
          {t('common.errorPrefix', { error })}
        </p>
      )
    }
    return null
  }

  return (
    <div className="page-viewer page-history-page">
      {page && !error ? (
        <div className="page-viewer__body page-history-page__body">
          <article className="page-history-page__content">
            <PageHistoryContent
              pageId={page.id}
              pageTitle={page.title}
              pageSlug={page.slug}
              testidPrefix="page-history-page"
            />
          </article>
        </div>
      ) : (
        renderError()
      )}
    </div>
  )
}
