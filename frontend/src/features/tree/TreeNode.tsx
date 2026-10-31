// 渲染可折叠树行、导航链接和按权限显示的节点菜单，并注册拖放所需的行引用。
import { TreeViewActionButton } from '@/features/tree/TreeViewActionButton'
import { NODE_KIND_SECTION, PageNode } from '@/shared/lib/api/pages'
import {
  DIALOG_ADD_PAGE,
  DIALOG_DELETE_PAGE_CONFIRMATION,
} from '@/shared/lib/registries/instances'
import { createNavigationVisitState } from '@/shared/lib/navigationVisit'
import { useIsMobile } from '@/shared/lib/useIsMobile'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useSessionStore } from '@/shared/state/session'
import { useTreeStore } from '@/features/tree/state/tree'
import { useDraggable, useDroppable } from '@dnd-kit/core'
import clsx from 'clsx'
import { ChevronUp, FilePlus, FolderPlus } from 'lucide-react'
import React, { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { useTreeDndStore } from '@/features/tree/treeDndStore'
import { useTreeNodeActionsMenusStore } from '@/features/tree/treeNodeActionsMenus'
import TreeNodeActionsMenu from '@/features/tree/TreeNodeActionsMenu'

type Props = {
  node: PageNode
  // 上级分区闭合时子行仍保留在结构中，但不注册拖放目标以避免测量不可见行。
  inClosedBranch?: boolean
}

export const TreeNode = React.memo(function TreeNode({
  node,
  inClosedBranch = false,
}: Props) {
  const { t } = useTranslation('viewer')
  const open = useTreeStore((s) => !!s.openNodeIdSet?.[node.id])
  const isStoreActive = useTreeStore((s) => s.activeNodeId === node.id)
  const toggleNode = useTreeStore((s) => s.toggleNode)
  const hasChildren = node.children && node.children.length > 0
  const openDialog = useDialogsStore((state) => state.openDialog)
  const isMobile = useIsMobile()
  const readOnlyMode = useIsReadOnly()
  const [hovered, setHovered] = useState(false)
  const isActionsMenuOpen = useTreeNodeActionsMenusStore(
    (s) => s.openMenuNodeId === node.id,
  )
  const isLoggedIn = useSessionStore((s) => s.user !== null)
  const isActive = isStoreActive
  // 删除确认对话框打开后菜单关闭，此状态继续标记对应行，直到对话框状态改变。
  const isDeleteTarget = useDialogsStore(
    (s) =>
      s.dialogType === DIALOG_DELETE_PAGE_CONFIRMATION &&
      s.dialogProps?.pageId === node.id,
  )

  const handleLinkClick = (event: React.MouseEvent<HTMLAnchorElement>) => {
    // 仅活动分区的普通左键点击用于折叠或展开，不拦截修饰键点击和其他节点导航。
    if (
      node.kind !== NODE_KIND_SECTION ||
      !isActive ||
      event.button !== 0 ||
      event.ctrlKey ||
      event.metaKey ||
      event.shiftKey ||
      event.altKey
    ) {
      return
    }

    toggleNode(node.id)
  }

  const dndEnabled = useTreeDndStore((s) => s.enabled)
  const isDragActive = useTreeDndStore((s) => s.activeId === node.id)
  const dropZone = useTreeDndStore((s) =>
    s.dropTarget?.nodeId === node.id ? s.dropTarget.zone : null,
  )
  const dndData = useMemo(() => ({ node }), [node])
  const dndDisabled = !dndEnabled || inClosedBranch
  // 拖放整体关闭或该行位于隐藏分支时，同时禁用拖动源与放置目标。
  const {
    setNodeRef: setDragRef,
    listeners,
    isDragging,
  } = useDraggable({
    id: node.id,
    data: dndData,
    disabled: dndDisabled,
  })
  const { setNodeRef: setDropRef } = useDroppable({
    id: node.id,
    data: dndData,
    disabled: dndDisabled,
  })
  const setRowRef = (el: HTMLElement | null) => {
    // 同一个 DOM 行同时作为可拖动项和可放置项注册。
    setDragRef(el)
    setDropRef(el)
  }

  const indent = 4
  const markerOffset = 8 // 左侧竖线距行边缘的像素位置。

  const linkText = (
    <div className={clsx('flex', 'tree-node__tooltip-parent')}>
      <Link
        to={`/${node.path}`}
        state={createNavigationVisitState()}
        className="tree-node__link"
        data-testid={`tree-node-link-${node.id}`}
        aria-current={isActive ? 'page' : undefined}
        draggable={false}
        onClick={handleLinkClick}
      >
        <span
          className={clsx('tree-node__title', {
            'tree-node__title--active': isActive,
            'tree-node__title--delete-target': isDeleteTarget,
          })}
        >
          {node.title || t('treeActions.untitledPage')}
        </span>
      </Link>
    </div>
  )

  const treeActionButtonStyle = isMobile ? '' : 'tree-node__actions--compact'

  return (
    <>
      <div
        ref={setRowRef}
        {...(dndEnabled ? listeners : {})}
        className={clsx('tree-node', {
          'tree-node--active': isActive,
          'tree-node--inactive': !isActive,
          'tree-node--dragging': isDragging,
          'tree-node--drop-inside': dropZone === 'inside',
          'tree-node--delete-target': isDeleteTarget,
        })}
        data-testid={`tree-node-${node.id}`}
        data-delete-target={isDeleteTarget ? 'true' : undefined}
        style={{ paddingLeft: indent }}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      >
        {dropZone === 'before' && (
          <div className="tree-node__drop-line tree-node__drop-line--top" />
        )}
        {dropZone === 'after' && (
          <div className="tree-node__drop-line tree-node__drop-line--bottom" />
        )}
        {dropZone === 'inside' && node.kind !== NODE_KIND_SECTION && (
          // 目标为普通页面时显示嵌套提示，完成放置后服务端将页面转换为分区。
          <FolderPlus size={14} className="tree-node__nest-hint" />
        )}
        <div
          className={clsx('tree-node__marker', {
            'tree-node__marker--active': isActive,
            'tree-node__marker--delete-target': isDeleteTarget,
          })}
          style={{ left: markerOffset }}
        />

        <div className="tree-node__main">
          {node.kind === NODE_KIND_SECTION && (
            <ChevronUp
              data-testid={`tree-node-toggle-icon-${node.id}`}
              size={16}
              className={clsx('tree-node__toggle', {
                'tree-node__toggle--open': open,
                'tree-node__toggle--closed': !open,
              })}
              onClick={() =>
                node.kind === NODE_KIND_SECTION && toggleNode(node.id)
              }
            />
          )}
          {
            // 普通页面没有展开图标，保留占位宽度与分区标题对齐。
            node.kind !== NODE_KIND_SECTION && (
              <div className="tree-node__toggle-spacer" />
            )
          }
          {linkText}
          {(isMobile || hovered || isActionsMenuOpen) && (
            <div className={clsx('tree-node__actions', treeActionButtonStyle)}>
              {!readOnlyMode && (
                <TreeViewActionButton
                  actionName="add"
                  icon={
                    <FilePlus
                      size={18}
                      className={clsx(
                        'tree-node__action-icon',
                        isMobile && 'text-brand/70!',
                      )}
                    />
                  }
                  tooltip={t('treeActions.createNewPageTooltip')}
                  onClick={() =>
                    openDialog(DIALOG_ADD_PAGE, { parentId: node.id })
                  }
                />
              )}
              {(!readOnlyMode || isLoggedIn) && (
                <TreeNodeActionsMenu node={node} />
              )}
            </div>
          )}
        </div>
      </div>

      <div
        className={clsx('tree-node__children', {
          'tree-node__children--closed': !open,
          'tree-node__children--dragging': isDragActive,
        })}
      >
        {hasChildren &&
          node.children?.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              inClosedBranch={inClosedBranch || !open}
            />
          ))}
      </div>
    </>
  )
})
