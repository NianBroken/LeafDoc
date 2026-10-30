import { Accordion } from '@/shared/components/ui/accordion'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/shared/components/ui/dropdown-menu'
import { FavoritesSection } from '@/features/favorites/FavoritesSection'
import { SidebarAccordionSection } from '@/features/sidebar/SidebarAccordionSection'
import { TreeViewActionButton } from '@/features/tree/TreeViewActionButton'
import { ApiLocalizedError, mapApiError } from '@/shared/lib/api/errors'
import { NODE_KIND_PAGE, NODE_KIND_SECTION } from '@/shared/lib/api/pages'
import {
  DIALOG_ADD_PAGE,
  DIALOG_SORT_PAGES,
} from '@/shared/lib/registries/instances'
import { buildViewUrl } from '@/shared/lib/routePath'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { toWikiLookupPath } from '@/shared/lib/wikiPath'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useFavoritesStore } from '@/features/favorites/state/favorites'
import { useResyncStore } from '@/features/settings/state/resync'
import { useSessionStore } from '@/shared/state/session'
import { useSidebarPanelsStore } from '@/features/sidebar/state/sidebarPanels'
import { useTreeStore } from '@/features/tree/state/tree'
import {
  ChevronsDown,
  ChevronsUp,
  FilePlus,
  FolderPlus,
  List,
  Loader2,
  MoreHorizontal,
  Pin,
  RefreshCw,
  Star,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation } from 'react-router'
import { toast } from 'sonner'
import { usePageEditorStore } from '@/features/editor/pageEditorStore'
import { PinnedSection } from '@/features/tree/PinnedSection'
import { TreeDndProvider } from '@/features/tree/TreeDnd'
import { TreeNode } from '@/features/tree/TreeNode'

export default function TreeView() {
  const { t } = useTranslation('viewer')
  const tree = useTreeStore((s) => s.tree)
  const loading = useTreeStore((s) => s.loading)
  const error = useTreeStore((s) => s.error)
  const { pathname } = useLocation()
  const reloadTree = useTreeStore((s) => s.reloadTree)
  const openAncestorsForPath = useTreeStore((s) => s.openAncestorsForPath)
  const setActiveNodeId = useTreeStore((s) => s.setActiveNodeId)
  const openNode = useTreeStore((s) => s.openNode)
  const expandAll = useTreeStore((s) => s.expandAll)
  const collapseAll = useTreeStore((s) => s.collapseAll)
  const appMode = useAppMode()
  const currentEditorPageId = usePageEditorStore(
    (state) => state.page?.id ?? state.initialPage?.id,
  )

  const currentPath = toWikiLookupPath(buildViewUrl(pathname))

  const pinnedPages = useTreeStore((s) => s.pinnedPages)
  const hasPinned = pinnedPages.length > 0
  const favoritePageIds = useFavoritesStore((s) => s.favoritePageIds)
  const user = useSessionStore((s) => s.user)
  const isLoggedIn = user !== null
  const isAdmin = user?.role === 'admin'
  const hasFavorites = isLoggedIn && favoritePageIds.size > 0
  const openDialog = useDialogsStore((state) => state.openDialog)
  const readOnlyMode = useIsReadOnly()
  const openSections = useSidebarPanelsStore((s) => s.openSections)
  const setOpenSections = useSidebarPanelsStore((s) => s.setOpenSections)
  const [refreshing, setRefreshing] = useState(false)

  useEffect(() => {
    if (!tree || !currentPath) return
    openAncestorsForPath(currentPath)
  }, [tree, currentPath, openAncestorsForPath])

  useEffect(() => {
    if (!tree) return
    if (appMode === 'edit' && currentEditorPageId) {
      openNode(currentEditorPageId)
      setActiveNodeId(currentEditorPageId)
      return
    }

    if (!currentPath) {
      setActiveNodeId(null)
      return
    }

    const node = useTreeStore.getState().getPageByPath(currentPath)
    setActiveNodeId(node?.id ?? null)
  }, [
    tree,
    appMode,
    currentEditorPageId,
    currentPath,
    openNode,
    setActiveNodeId,
  ])

  useEffect(() => {
    if (tree === null) {
      reloadTree()
    }
  }, [tree, reloadTree])

  const handleRefresh = async () => {
    if (!isAdmin) {
      await reloadTree()
      return
    }

    setRefreshing(true)
    try {
      await useResyncStore.getState().trigger()
      await reloadTree()
      // reloadTree() catches its own fetch failures and only records them in
      // useTreeStore's error state — it never rejects — so a failed final
      // reload must be detected here rather than via this try's catch.
      if (useTreeStore.getState().error) {
        toast.error(t('toolbar.refreshError'))
      } else {
        toast.success(t('toolbar.refreshSuccess'))
      }
    } catch (err) {
      if (
        err instanceof ApiLocalizedError &&
        err.code === 'resync_already_running'
      ) {
        toast.info(t('toolbar.refreshAlreadyRunning'))
      } else {
        toast.error(mapApiError(err, t('toolbar.refreshError')).message)
      }
    } finally {
      setRefreshing(false)
    }
  }

  if (loading)
    return (
      <p className="tree-view__status tree-view__status--loading">
        {t('treeActions.loading')}
      </p>
    )

  if (error || !tree)
    return (
      <p className="tree-view__status tree-view__status--error">
        {t('treeActions.loadErrorPrefix', { error })}
      </p>
    )

  const pagesToolbar = (
    <>
      {!readOnlyMode && (
        <>
          <TreeViewActionButton
            actionName="add"
            icon={
              <FilePlus
                className="tree-view__action-icon text-brand/70!"
                size={18}
              />
            }
            tooltip={t('treeActions.createNewPageTooltip')}
            onClick={() =>
              openDialog(DIALOG_ADD_PAGE, {
                parentId: '',
                nodeKind: NODE_KIND_PAGE,
              })
            }
          />
          <TreeViewActionButton
            actionName="add-section"
            icon={
              <FolderPlus
                className="tree-view__action-icon text-brand/70!"
                size={18}
              />
            }
            tooltip={t('treeActions.createNewSectionTooltip')}
            onClick={() =>
              openDialog(DIALOG_ADD_PAGE, {
                parentId: '',
                nodeKind: NODE_KIND_SECTION,
              })
            }
          />
        </>
      )}
      <TreeViewActionButton
        actionName="refresh"
        icon={
          refreshing ? (
            <Loader2
              className="tree-view__action-icon animate-spin"
              size={18}
            />
          ) : (
            <RefreshCw className="tree-view__action-icon" size={18} />
          )
        }
        tooltip={t('toolbar.refresh')}
        onClick={handleRefresh}
        disabled={refreshing}
      />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <TreeViewActionButton
            actionName="tree-overflow"
            icon={
              <MoreHorizontal className="tree-view__action-icon" size={18} />
            }
            tooltip={t('toolbar.moreActions')}
          />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-44">
          <DropdownMenuItem
            className="cursor-pointer gap-2"
            onClick={expandAll}
            data-testid="tree-view-action-button-expand-all"
          >
            <ChevronsDown size={15} />
            {t('toolbar.expandAll')}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="cursor-pointer gap-2"
            onClick={collapseAll}
            data-testid="tree-view-action-button-collapse-all"
          >
            <ChevronsUp size={15} />
            {t('toolbar.collapseAll')}
          </DropdownMenuItem>
          {!readOnlyMode && tree && (
            <DropdownMenuItem
              className="cursor-pointer gap-2"
              onClick={() => openDialog(DIALOG_SORT_PAGES, { parent: tree })}
              data-testid="tree-view-action-button-sort"
            >
              <List size={15} />
              {t('toolbar.sortPages')}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  )

  return (
    <Accordion
      type="multiple"
      value={openSections}
      onValueChange={setOpenSections}
      className="tree-view"
    >
      {hasPinned && (
        <SidebarAccordionSection
          value="pinned"
          title={t('pinned.sectionTitle')}
          icon={<Pin size={11} />}
          collapseToggleLabel={t('pinned.togglePinnedSection')}
        >
          <PinnedSection />
        </SidebarAccordionSection>
      )}
      {hasFavorites && (
        <SidebarAccordionSection
          value="favorites"
          title={t('favorites.sectionTitle')}
          icon={<Star size={11} />}
          collapseToggleLabel={t('favorites.toggleFavoritesSection')}
        >
          <FavoritesSection />
        </SidebarAccordionSection>
      )}
      <SidebarAccordionSection
        value="pages"
        title={t('pinned.pagesSectionTitle')}
        collapseToggleLabel={t('pinned.togglePagesSection')}
        actions={pagesToolbar}
      >
        <TreeDndProvider enabled={!readOnlyMode}>
          <div className="tree-view__nodes">
            {tree?.children?.map((node) => (
              <TreeNode key={node.id} node={node} />
            ))}
          </div>
        </TreeDndProvider>
      </SidebarAccordionSection>
    </Accordion>
  )
}
