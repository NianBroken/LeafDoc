// 提供单个树节点的操作菜单，按权限与节点种类显示增删改、移动、固定和收藏动作。
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/components/ui/dropdown-menu'
import {
  applyPageRefactor,
  convertPage,
  getPageByPath,
  NODE_KIND_PAGE,
  NODE_KIND_SECTION,
  pinPage,
  previewPageRefactor,
  updatePage,
} from '@/shared/lib/api/pages'
import type { Page, PageNode } from '@/shared/lib/api/pages'
import { asApiLocalizedError, mapApiError } from '@/shared/lib/api/errors'
import {
  DIALOG_ADD_PAGE,
  DIALOG_COPY_PAGE,
  DIALOG_DELETE_PAGE_CONFIRMATION,
  DIALOG_EDIT_PAGE_METADATA,
  DIALOG_MOVE_PAGE,
  DIALOG_SORT_PAGES,
} from '@/shared/lib/registries/instances'
import i18next from '@/shared/lib/i18n'
import { stripBasePath } from '@/shared/lib/routePath'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { getDeleteRedirectRoutePath } from '@/shared/lib/wikiPath'
import { useConfigStore } from '@/shared/state/config'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useFavoritesStore } from '@/features/favorites/state/favorites'
import { useSessionStore } from '@/shared/state/session'
import { useViewerStore } from '@/features/viewer/viewer'
import { useTreeStore } from '@/features/tree/state/tree'
import {
  Copy,
  FilePlus,
  FolderPlus,
  List,
  MoreVertical,
  Move,
  Pencil,
  Pin,
  PinOff,
  Repeat2,
  Star,
  Trash,
} from 'lucide-react'
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { toast } from 'sonner'
import { usePageEditorStore } from '@/features/editor/pageEditorStore'
import { confirmPageRefactor } from '@/features/page/pageRefactorDialogState'
import { TreeViewActionButton } from '@/features/tree/TreeViewActionButton'
import { useTreeNodeActionsMenusStore } from '@/features/tree/treeNodeActionsMenus'

export type TreeNodeActionsMenuProps = {
  node: PageNode
}

export default function TreeNodeActionsMenu({
  node,
}: TreeNodeActionsMenuProps) {
  const { t } = useTranslation('viewer')
  const { id: nodeId, kind: nodeKind, children, version: nodeVersion } = node
  const currentEditorPageId = usePageEditorStore((state) => state.page?.id)
  const enableLinkRefactor = useConfigStore((s) => s.enableLinkRefactor)
  const openDialog = useDialogsStore((state) => state.openDialog)
  const reloadTree = useTreeStore((state) => state.reloadTree)
  const readOnlyMode = useIsReadOnly()
  const isLoggedIn = useSessionStore((s) => s.user !== null)
  const isFavorited = useFavoritesStore((s) => s.favoritePageIds.has(nodeId))
  const addFavorite = useFavoritesStore((s) => s.addFavorite)
  const removeFavorite = useFavoritesStore((s) => s.removeFavorite)
  const hasChildren = children && children.length > 0
  // 子节点检查用于菜单入口显隐，分区与页面名称由实际节点种类取得。
  const navigate = useNavigate()
  const setOpenMenuNodeId = useTreeNodeActionsMenusStore(
    (s) => s.setOpenMenuNodeId,
  )
  const open = useTreeNodeActionsMenusStore((s) => s.openMenuNodeId === node.id)
  const itemLabel =
    nodeKind === NODE_KIND_PAGE
      ? i18next.t('common.page', { ns: 'page' })
      : i18next.t('common.section', { ns: 'page' })
  const itemLabelCapitalized =
    nodeKind === NODE_KIND_PAGE
      ? i18next.t('common.pageCapitalized', { ns: 'page' })
      : i18next.t('common.sectionCapitalized', { ns: 'page' })

  const handleConvertPage = useCallback(() => {
    // 仅在空分区入口调用转换；版本冲突时刷新树并在当前查看同一节点时重新读取页面。
    convertPage(
      nodeId,
      nodeKind === NODE_KIND_PAGE ? NODE_KIND_SECTION : NODE_KIND_PAGE,
      nodeVersion,
    )
      .then(() => {
        toast.success(t('treeActions.convertedToast'))
        reloadTree()
      })
      .catch((err) => {
        const localized = asApiLocalizedError(err)
        if (localized?.code === 'page_version_conflict') {
          reloadTree()
          const viewerPage = useViewerStore.getState().page
          if (viewerPage?.id === nodeId && viewerPage.path) {
            useViewerStore
              .getState()
              .loadPageData(viewerPage.path)
              .catch(console.error)
          }
          toast.error(t('treeActions.conflictRetryMessage'))
        } else {
          const mapped = mapApiError(err, t('treeActions.convertErrorFallback'))
          toast.error(mapped.message)
        }
      })
  }, [nodeId, nodeKind, nodeVersion, reloadTree, t])

  const setPinnedLocally = useTreeStore((s) => s.setPinnedLocally)

  const getCurrentRoutePath = useCallback(() => {
    if (typeof window === 'undefined') {
      return '/'
    }
    return window.location.pathname
  }, [])

  const handleTogglePin = useCallback(() => {
    // 固定请求携带节点版本，接口成功后用返回版本更新本地树，失败显示映射后的错误文本。
    const newPinned = !node.pinned
    pinPage(nodeId, nodeVersion, newPinned)
      .then((updated) => {
        setPinnedLocally(nodeId, newPinned, updated.version)
        toast.success(
          node.pinned ? t('pinned.unpinSuccess') : t('pinned.pinSuccess'),
        )
      })
      .catch((err) =>
        toast.error(mapApiError(err, t('pinned.pinError')).message),
      )
  }, [nodeId, nodeVersion, node.pinned, setPinnedLocally, t])

  const handleToggleFavorite = useCallback(async () => {
    // 收藏动作复用乐观状态容器，接口失败由容器回滚并在此处显示错误提示。
    try {
      if (isFavorited) {
        await removeFavorite(nodeId)
        toast.success(t('favorites.removeSuccess'))
      } else {
        await addFavorite(nodeId)
        toast.success(t('favorites.addSuccess'))
      }
    } catch (err) {
      toast.error(mapApiError(err, t('favorites.favoriteError')).message)
    }
  }, [isFavorited, addFavorite, removeFavorite, nodeId, t])

  const handleRenamePage = useCallback(
    async (title: string, slug: string) => {
      // 编辑器正在编辑目标节点时拒绝从菜单改名，避免覆盖编辑页面中的当前状态。
      if (currentEditorPageId === nodeId) {
        toast.warning(
          t('treeActions.renameEditingWarning', { item: itemLabel }),
        )
        return
      }

      try {
        const page = await getPageByPath(node.path)
        const titleChanged = page.title !== title
        const slugChanged = page.slug !== slug

        if (!titleChanged && !slugChanged) {
          // 标题与路径段都未改变时不调用更新接口，也不刷新树。
          return
        }

        let updatedPage: Page | null

        if (slugChanged && enableLinkRefactor) {
          // 仅路径段变化且功能开关开启时显示影响预览，取消返回空值并终止提交。
          const preview = await previewPageRefactor(page.id, {
            kind: 'rename',
            title,
            slug,
          })
          const rewriteLinks = await confirmPageRefactor(preview, {
            allowSkipRewrite: true,
          })

          if (rewriteLinks === null) {
            return
          }

          updatedPage = await applyPageRefactor(page.id, {
            kind: 'rename',
            version: page.version,
            title,
            slug,
            content: page.content,
            rewriteLinks,
          })
        } else {
          updatedPage = await updatePage(
            page.id,
            page.version,
            title,
            slug,
            page.content,
            page.tags ?? [],
            page.properties ?? {},
          )
        }

        await reloadTree()

        const viewerPage = useViewerStore.getState().page
        // 当前查看页存在时合并接口数据，并保留响应中未提供的标签和属性字段。
        if (viewerPage?.id === nodeId && updatedPage) {
          useViewerStore.setState({
            page: {
              ...viewerPage,
              ...updatedPage,
              tags: updatedPage.tags ?? page.tags ?? [],
              properties: updatedPage.properties ?? page.properties ?? {},
            },
            notFound: false,
            error: null,
          })
        }

        const currentRoutePath = getCurrentRoutePath()
        const currentRouterPath =
          stripBasePath(currentRoutePath) ?? currentRoutePath
        if (currentRouterPath === `/${node.path}` && updatedPage?.path) {
          // 只有浏览器当前正位于原节点路径时才导航到新路径。
          navigate(`/${updatedPage.path}`)
        }

        toast.success(
          t('treeActions.renamedToast', { item: itemLabelCapitalized }),
        )
      } catch (err) {
        // 版本冲突刷新树并提示重试，其余错误统一映射后显示。
        const localized = asApiLocalizedError(err)
        if (localized?.code === 'page_version_conflict') {
          await reloadTree()
          toast.error(t('treeActions.conflictRetryMessage'))
          return
        }

        const mapped = mapApiError(err, t('treeActions.renameErrorFallback'))
        toast.error(mapped.message)
      }
    },
    [
      currentEditorPageId,
      enableLinkRefactor,
      getCurrentRoutePath,
      navigate,
      node.path,
      nodeId,
      reloadTree,
      t,
      itemLabel,
      itemLabelCapitalized,
    ],
  )

  return (
    <DropdownMenu
      open={open}
      onOpenChange={(nextOpen) => setOpenMenuNodeId(nextOpen ? node.id : null)}
    >
      <DropdownMenuTrigger asChild aria-label={t('toolbar.moreActions')}>
        <TreeViewActionButton
          actionName="open-more-actions"
          icon={<MoreVertical size={18} className="tree-node__action-icon" />}
          tooltip={t('toolbar.moreActions')}
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        {/* 只读环境隐藏编辑操作，登录者仍可使用收藏入口。 */}
        {!readOnlyMode && (
          <>
            <DropdownMenuItem
              className="cursor-pointer"
              onClick={() => {
                openDialog(DIALOG_ADD_PAGE, {
                  parentId: nodeId,
                  nodeKind: NODE_KIND_PAGE,
                })
              }}
            >
              <FilePlus size={18} className="tree-node__action-icon" />{' '}
              {t('treeActions.menuAddPage')}
            </DropdownMenuItem>
            <DropdownMenuItem
              className="cursor-pointer"
              onClick={() => {
                openDialog(DIALOG_ADD_PAGE, {
                  parentId: nodeId,
                  nodeKind: NODE_KIND_SECTION,
                })
              }}
            >
              <FolderPlus size={18} className="tree-node__action-icon" />{' '}
              {t('treeActions.menuAddSection')}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="cursor-pointer"
              onClick={() => {
                navigate(`/e/${node.path}`)
              }}
            >
              <Pencil size={18} className="tree-node__action-icon" />{' '}
              {t('treeActions.menuEdit', { item: itemLabelCapitalized })}
            </DropdownMenuItem>
            <DropdownMenuItem
              className="cursor-pointer"
              data-testid="tree-view-action-button-rename"
              onClick={() => {
                openDialog(DIALOG_EDIT_PAGE_METADATA, {
                  parentId: node.parentId ?? '',
                  currentId: node.id,
                  itemKind: node.kind,
                  title: node.title,
                  slug: node.slug,
                  onChange: handleRenamePage,
                })
              }}
            >
              <Pencil size={18} className="tree-node__action-icon" />{' '}
              {t('treeActions.menuRename', { item: itemLabelCapitalized })}
            </DropdownMenuItem>
            {nodeKind === NODE_KIND_PAGE && (
              <DropdownMenuItem
                className="cursor-pointer"
                onClick={() => {
                  openDialog(DIALOG_COPY_PAGE, { sourcePage: node })
                }}
              >
                <Copy size={18} className="tree-node__action-icon" />{' '}
                {t('treeActions.menuCopyPage')}
              </DropdownMenuItem>
            )}
            {hasChildren && (
              <DropdownMenuItem
                className="cursor-pointer"
                data-testid="tree-view-action-button-sort"
                onClick={() => openDialog(DIALOG_SORT_PAGES, { parent: node })}
              >
                <List size={18} className="tree-node__action-icon" />{' '}
                {t('treeActions.menuSortChildren', {
                  item: itemLabelCapitalized,
                })}
              </DropdownMenuItem>
            )}
            <DropdownMenuItem
              className="cursor-pointer"
              data-testid="tree-view-action-button-move"
              onClick={() => openDialog(DIALOG_MOVE_PAGE, { pageId: node.id })}
            >
              <Move size={18} className="tree-node__action-icon" />{' '}
              {t('treeActions.menuMove', { item: itemLabelCapitalized })}
            </DropdownMenuItem>
            {nodeKind === NODE_KIND_SECTION && !hasChildren && (
              <DropdownMenuItem
                className="cursor-pointer"
                onClick={handleConvertPage}
              >
                <Repeat2 size={18} className="tree-node__action-icon" />{' '}
                {t('treeActions.menuConvertToPage')}
              </DropdownMenuItem>
            )}
            <DropdownMenuSeparator />
          </>
        )}
        {isLoggedIn && (
          <DropdownMenuItem
            className="cursor-pointer"
            data-testid="tree-view-action-button-favorite"
            onClick={handleToggleFavorite}
          >
            <Star
              size={18}
              className="tree-node__action-icon"
              fill={isFavorited ? 'currentColor' : 'none'}
            />{' '}
            {isFavorited
              ? t('favorites.removeFavorite')
              : t('favorites.addFavorite')}
          </DropdownMenuItem>
        )}
        {!readOnlyMode && (
          <DropdownMenuItem
            className="cursor-pointer"
            data-testid="tree-view-action-button-pin"
            onClick={handleTogglePin}
          >
            {node.pinned ? (
              <>
                <PinOff size={18} className="tree-node__action-icon" />{' '}
                {t('pinned.unpinPage')}
              </>
            ) : (
              <>
                <Pin size={18} className="tree-node__action-icon" />{' '}
                {t('pinned.pinPage')}
              </>
            )}
          </DropdownMenuItem>
        )}
        {!readOnlyMode && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-error cursor-pointer"
              data-testid="tree-view-action-button-delete"
              onClick={() => {
                const currentRoutePath = getCurrentRoutePath()
                const currentRouterPath =
                  stripBasePath(currentRoutePath) ?? currentRoutePath
                const isCurrentlyEditedNode =
                  currentRouterPath.startsWith('/e/') &&
                  currentEditorPageId === node.id

                if (isCurrentlyEditedNode) {
                  // 禁止从删除菜单移除当前编辑节点，其他节点打开确认对话框并计算离开路径。
                  toast.warning(
                    t('treeActions.deleteEditingWarning', { item: itemLabel }),
                  )
                  return
                }

                openDialog(DIALOG_DELETE_PAGE_CONFIRMATION, {
                  pageId: node?.id,
                  redirectTo: getDeleteRedirectRoutePath(
                    currentRouterPath,
                    node.path,
                  ),
                })
              }}
            >
              <Trash size={18} className="tree-node__action-icon text-error" />{' '}
              {t('treeActions.menuDelete', { item: itemLabelCapitalized })}
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
