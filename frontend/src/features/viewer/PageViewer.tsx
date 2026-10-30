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
import {
  getPageAttachments,
  type PageAttachment,
} from '@/shared/lib/api/assets'
import { NODE_KIND_SECTION } from '@/shared/lib/api/pages'
import { mapApiError } from '@/shared/lib/api/errors'
import { firstDescendantPage } from '@/features/viewer/sectionNavigation'
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
import { useTocPanelStore } from '@/features/preview/state/tocPanel'
import { useTreeStore } from '@/features/tree/state/tree'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
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
  const tree = useTreeStore((s) => s.tree)
  const treeLoading = useTreeStore((s) => s.loading)
  const treeError = useTreeStore((s) => s.error)
  const loading = useViewerStore((s) => s.isLoading)
  const error = useViewerStore((s) => s.error)
  const notFound = useViewerStore((s) => s.notFound)
  const page = useViewerStore((s) => s.page)
  const loadPageData = useViewerStore((s) => s.loadPageData)
  const clearViewer = useViewerStore((s) => s.clear)

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

  const sectionNode = useTreeStore((s) => (page ? s.byId[page.id] : undefined))
  const emptySection = page?.kind === NODE_KIND_SECTION && !page.content.trim()
  const sectionTarget =
    emptySection && sectionNode ? firstDescendantPage(sectionNode) : null
  useEffect(() => {
    // Resolve only a current, fully loaded section; the normal viewer fetch checks the target.
    if (
      !emptySection ||
      !sectionTarget ||
      loading ||
      treeLoading ||
      treeError ||
      error
    )
      return
    if (toWikiLookupPath(pathname) !== toWikiLookupPath(page.path)) return
    navigate(getWikiTargetRoutePath(sectionTarget.path), {
      replace: true,
      state: createNavigationVisitState(),
    })
  }, [
    emptySection,
    sectionTarget,
    loading,
    treeLoading,
    treeError,
    error,
    pathname,
    page,
    navigate,
    tree,
  ])

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
  const [attachmentError, setAttachmentError] = useState<string | null>(null)

  useEffect(() => {
    if (!page?.id) {
      setAttachments([])
      return
    }

    let cancelled = false
    setAttachmentError(null)
    getPageAttachments(page.id)
      .then((files) => {
        if (!cancelled) setAttachments(files)
      })
      .catch((err) => {
        if (!cancelled) {
          setAttachments([])
          setAttachmentError(
            mapApiError(
              err,
              t('failed to load page attachments', { ns: 'errors' }),
            ).message,
          )
        }
      })

    return () => {
      cancelled = true
    }
  }, [page?.id, t])

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
              {attachmentError && (
                <p role="alert" className="settings__notice">
                  {attachmentError}
                </p>
              )}
              {emptySection && treeError && (
                <p role="alert" className="settings__notice">
                  {treeError}
                </p>
              )}
              {!treeLoading && !treeError && !sectionTarget && (
                <EmptySectionChildrenList page={page} />
              )}
            </article>
            <BacklinkInfo />
          </div>
        )}
        {renderError()}
      </div>
    </>
  )
}
