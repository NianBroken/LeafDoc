import Page404 from '@/shared/components/Page404'
import { useIsMobile } from '@/shared/lib/useIsMobile'
import { formatRelativeTime } from '@/shared/lib/formatDate'
import {
  createNavigationVisitState,
  getNavigationVisitKey,
} from '@/shared/lib/navigationVisit'
import {
  DIALOG_COPY_PAGE,
  DIALOG_DELETE_PAGE_CONFIRMATION,
  DIALOG_PAGE_PERMALINK,
} from '@/shared/lib/registries/instances'
import { buildHistoryUrl } from '@/shared/lib/routePath'
import { FavoriteToggleButton } from '@/features/favorites/FavoriteToggleButton'
import {
  getPageAttachments,
  type PageAttachment,
} from '@/shared/lib/api/assets'
import { pinPage } from '@/shared/lib/api/pages'
import { createHotkeyDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { useScrollRestoration } from '@/shared/lib/useScrollRestoration'
import { cn } from '@/shared/lib/utils'
import {
  getParentWikiRoutePath,
  getWikiTargetRoutePath,
  toWikiLookupPath,
} from '@/shared/lib/wikiPath'
import { useConfigStore } from '@/shared/state/config'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useHotKeysStore } from '@/shared/state/hotkeys'
import { useSessionStore } from '@/shared/state/session'
import { useTocPanelStore } from '@/features/preview/state/tocPanel'
import { useTreeStore } from '@/features/tree/state/tree'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createPortal } from 'react-dom'
import { useLocation, useNavigate } from 'react-router'
import { BacklinkInfo } from '@/features/links/LinkInfo'
import { extractTocEntries } from '@/features/preview/extractTocEntries'
import MarkdownPreview from '@/features/preview/MarkdownPreview'
import { TocDropdownButton } from '@/features/preview/TocDropdownButton'
import { TocSidePanel } from '@/features/preview/TocSidePanel'
import { isTocInitiallyCollapsed } from '@/features/preview/tocVisibility'
import { useTocScrollSpy } from '@/features/preview/useTocScrollSpy'
import Breadcrumbs from '@/features/viewer/Breadcrumbs'
import EmptySectionChildrenList from '@/features/viewer/EmptySectionChildrenList'
import { PageMetadata } from '@/features/viewer/PageMetadata'
import { useScrollToHeadline } from '@/features/viewer/useScrollToHeadline'
import { useScrollToSearchQuery } from '@/features/viewer/useScrollToSearchQuery'
import { useSetPageTitle } from '@/features/viewer/useSetPageTitle'
import { useToolbarActions } from '@/features/viewer/useToolbarActions'
import { useViewerStore } from '@/features/viewer/viewer'

function displayUser(label?: { username: string }) {
  return label?.username || null
}

export default function PageViewer() {
  const isMobile = useIsMobile()
  const { t } = useTranslation('viewer')
  const location = useLocation()
  const { pathname } = location
  const navigate = useNavigate()
  const openDialog = useDialogsStore((state) => state.openDialog)
  const registerHotkey = useHotKeysStore((state) => state.registerHotkey)
  const unregisterHotkey = useHotKeysStore((state) => state.unregisterHotkey)
  const openNode = useTreeStore((state) => state.openNode)
  const setPinnedLocally = useTreeStore((s) => s.setPinnedLocally)
  const loading = useViewerStore((s) => s.isLoading)
  const error = useViewerStore((s) => s.error)
  const notFound = useViewerStore((s) => s.notFound)
  const page = useViewerStore((s) => s.page)
  const loadPageData = useViewerStore((s) => s.loadPageData)
  const clearViewer = useViewerStore((s) => s.clear)
  const isPinned = useTreeStore((s) =>
    page ? (s.byId[page.id]?.pinned ?? false) : false,
  )
  const isLoggedIn = useSessionStore((s) => s.user !== null)

  const actions = {
    pageKind: page?.kind,
    printPage: useCallback(() => {
      window.print()
    }, []),
    editPage: useCallback(() => {
      clearViewer()
      navigate(`/e/${page?.path || ''}`)
    }, [page, navigate, clearViewer]),
    showHistory: useCallback(() => {
      navigate(buildHistoryUrl(page?.path || pathname), {
        state: createNavigationVisitState(),
      })
    }, [navigate, page, pathname]),
    showPermalink: useCallback(() => {
      if (!page) return
      openDialog(DIALOG_PAGE_PERMALINK, { page })
    }, [page, openDialog]),
    deletePage: useCallback(() => {
      openDialog(DIALOG_DELETE_PAGE_CONFIRMATION, {
        pageId: page?.id,
        redirectTo: getParentWikiRoutePath(page?.path || '/'),
      })
    }, [page, openDialog]),
    copyPage: useCallback(() => {
      if (!page) return
      openDialog(DIALOG_COPY_PAGE, { sourcePage: page })
    }, [page, openDialog]),
    isPinned,
    onPinToggle: useCallback(() => {
      if (!page) return
      const newPinned = !isPinned
      pinPage(page.id, page.version, newPinned)
        .then((updated) => {
          setPinnedLocally(page.id, newPinned, updated.version)
          toast.success(
            isPinned ? t('pinned.unpinSuccess') : t('pinned.pinSuccess'),
          )
        })
        .catch(() => toast.error(t('pinned.pinError')))
    }, [page, isPinned, setPinnedLocally, t]),
  }

  useScrollRestoration(getNavigationVisitKey(location), loading)
  useScrollToHeadline({ content: page?.content || '', isLoading: loading })
  useScrollToSearchQuery({ content: page?.content || '', isLoading: loading })
  useToolbarActions(actions)
  useSetPageTitle({ page })

  useEffect(() => {
    const path = toWikiLookupPath(pathname)
    loadPageData?.(path)
  }, [pathname, loadPageData])

  useEffect(() => {
    if (!page?.id) return
    openNode(page.id)
  }, [openNode, page?.id])

  const renderError = () => {
    if (!loading && notFound) {
      return (
        <Page404 allowCreate targetPath={getWikiTargetRoutePath(pathname)} />
      )
    }
    if (!loading && error) {
      return (
        <p className="page-viewer__error">
          {t('treeActions.loadErrorPrefix', { error })}
        </p>
      )
    }
    return null
  }

  const tocEntries = useMemo(
    () => (page ? extractTocEntries(page.content) : []),
    [page],
  )
  const [attachments, setAttachments] = useState<PageAttachment[]>([])

  useEffect(() => {
    if (!page?.id) {
      setAttachments([])
      return
    }

    let cancelled = false
    getPageAttachments(page.id)
      .then((files) => {
        if (!cancelled) setAttachments(files)
      })
      .catch(() => {
        if (!cancelled) setAttachments([])
      })

    return () => {
      cancelled = true
    }
  }, [page?.id])

  const tocMode = useConfigStore((s) => s.tocMode)
  const showToc = tocEntries.length > 0
  useEffect(() => {
    useTocPanelStore
      .getState()
      .setCollapsed(isTocInitiallyCollapsed(tocEntries.length, tocMode))
  }, [page?.id, tocMode, tocEntries.length])
  const showRightPane = showToc || attachments.length > 0
  // Single scroll spy for both the dropdown and the side panel.
  const tocActiveId = useTocScrollSpy(showToc ? tocEntries : [])

  const toggleTocCollapsed = useTocPanelStore((state) => state.toggleCollapsed)
  const tocCollapsed = useTocPanelStore((state) => state.collapsed)

  useEffect(() => {
    if (!showRightPane) return

    const tocToggleHotkey = createHotkeyDefinition(
      'viewer.toc.toggle',
      toggleTocCollapsed,
    )
    registerHotkey(tocToggleHotkey)

    return () => unregisterHotkey(tocToggleHotkey)
  }, [showRightPane, toggleTocCollapsed, registerHotkey, unregisterHotkey])

  const editorName = displayUser(page?.metadata?.lastAuthor)
  const updatedRelative = formatRelativeTime(page?.metadata?.updatedAt)
  const showUpdated = updatedRelative
  const subheaderRoot = document.getElementById('app-subheader-root')
  const tocPaneRoot = document.getElementById('app-toc-pane-root')

  const subheader =
    page && !error && subheaderRoot
      ? createPortal(
          <div
            className={cn(
              'page-viewer__subheader print:hidden',
              showRightPane &&
                (tocCollapsed
                  ? 'page-viewer__subheader--toc-reserved-collapsed'
                  : 'page-viewer__subheader--toc-reserved'),
            )}
          >
            <div className="page-viewer__subheader-inner">
              <div className="page-viewer__subheader-main">
                <div className="page-viewer__subheader-copy">
                  <Breadcrumbs />
                  {showUpdated && (
                    <div className="page-viewer__metadata">
                      <span className="page-viewer__metadata-item">
                        {editorName
                          ? t('section.updatedByLabel', {
                              editor: editorName,
                              time: updatedRelative,
                            })
                          : t('section.updatedLabel', {
                              time: updatedRelative,
                            })}
                      </span>
                    </div>
                  )}
                </div>
                {isLoggedIn && (
                  <FavoriteToggleButton
                    pageId={page.id}
                    size={16}
                    className="page-viewer__favorite-toggle"
                  />
                )}
              </div>
              {showRightPane && (
                <div className="page-viewer__toc-button">
                  <TocDropdownButton
                    open={isMobile ? !tocCollapsed : false}
                    onOpenChange={
                      isMobile
                        ? (open) =>
                            useTocPanelStore.getState().setCollapsed(!open)
                        : undefined
                    }
                    entries={showToc ? tocEntries : []}
                    clickable
                    activeId={tocActiveId}
                    downloads={attachments}
                  />
                </div>
              )}
            </div>
          </div>,
          subheaderRoot,
        )
      : null

  const tocPane =
    showRightPane && page && !error && tocPaneRoot
      ? createPortal(
          <TocSidePanel
            entries={showToc ? tocEntries : []}
            activeId={tocActiveId}
            downloads={attachments}
          />,
          tocPaneRoot,
        )
      : null

  return (
    <>
      {subheader}
      {tocPane}
      {page && !error && (
        <div className="page-viewer__metadata-bar hidden sm:block print:hidden">
          <div className="page-viewer__metadata-bar-inner">
            <PageMetadata page={page} />
          </div>
        </div>
      )}
      <div className="page-viewer">
        {page && !error && (
          <div className="page-viewer__body">
            <article className="page-viewer__content">
              <MarkdownPreview content={page.content} path={page.path} />
              <EmptySectionChildrenList page={page} />
            </article>
            <BacklinkInfo />
          </div>
        )}
        {renderError()}
      </div>
    </>
  )
}
