import {
  convertPage,
  NODE_KIND_PAGE,
  NODE_KIND_SECTION,
  PageNode,
  sortPages,
} from '@/shared/lib/api/pages'
import { refreshAfterPageRefactor } from '@/features/page/pageMutationRefresh'
import { useTreeStore } from '@/features/tree/state/tree'
import {
  DndContext,
  DragEndEvent,
  DragMoveEvent,
  DragOverlay,
  DragStartEvent,
  Modifier,
  MouseSensor,
  TouchSensor,
  pointerWithin,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import { getEventCoordinates } from '@dnd-kit/utilities'
import { useCallback, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router'
import { toast } from 'sonner'
import i18next from '@/shared/lib/i18n'
import { performTreeCrossParentMove } from '@/features/tree/treeDndMove'
import { useTreeDndStore } from '@/features/tree/treeDndStore'
import {
  buildOrderedIds,
  collectSubtreeIds,
  DropResolution,
  DropTarget,
  getDropZone,
  resolveDrop,
  ROOT_ID,
} from '@/features/tree/treeDndUtils'

const EXPAND_ON_HOVER_DELAY_MS = 600

// 拖出最后一个子项后保留空分区，提示由用户决定是否转换回普通页面。
function offerConvertBackToPage(parentId: string) {
  const { byId } = useTreeStore.getState()
  const parent = byId[parentId]
  if (!parent || parent.kind !== NODE_KIND_SECTION) return
  if ((parent.children?.length ?? 0) > 0) return

  toast(
    i18next.t('treeActions.emptySectionToast', {
      ns: 'viewer',
      title: parent.title,
    }),
    {
      action: {
        label: i18next.t('treeActions.convertBackAction', { ns: 'viewer' }),
        onClick: () => {
          const node = useTreeStore.getState().byId[parentId]
          if (!node) return
          convertPage(parentId, NODE_KIND_PAGE, node.version)
            .then(() => useTreeStore.getState().reloadTree({ silent: true }))
            .then(() =>
              toast.success(
                i18next.t('treeActions.convertedBackToast', {
                  ns: 'viewer',
                  title: node.title,
                }),
              ),
            )
            .catch((err) => {
              console.warn(err)
              toast.error(
                i18next.t('treeActions.convertBackErrorFallback', {
                  ns: 'viewer',
                }),
              )
            })
        },
      },
    },
  )
}

// 将拖动预览定位到指针右侧 12 像素并在垂直方向居中，不沿用原行内偏移。
const followCursor: Modifier = ({
  activatorEvent,
  draggingNodeRect,
  transform,
}) => {
  if (!draggingNodeRect || !activatorEvent) {
    return transform
  }
  const coords = getEventCoordinates(activatorEvent)
  if (!coords) {
    return transform
  }
  return {
    ...transform,
    x: transform.x + coords.x - draggingNodeRect.left + 12,
    y:
      transform.y +
      coords.y -
      draggingNodeRect.top -
      draggingNodeRect.height / 2,
  }
}

function getPointerY(
  activatorEvent: Event,
  delta: { y: number },
): number | null {
  const mouseLike = activatorEvent as MouseEvent
  if (typeof mouseLike.clientY === 'number') {
    return mouseLike.clientY + delta.y
  }
  const touch = (activatorEvent as TouchEvent).touches?.[0]
  if (touch) {
    return touch.clientY + delta.y
  }
  return null
}

// 拖放后的点击可能落到被拖动的链接并触发整页导航，因此在窗口捕获阶段阻止一次点击。
// 监听器在拖动期间预先安装，首次捕获后自我移除，避免抑制同一拖动后的其他用户操作。
function suppressClickCapture(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()
  window.removeEventListener('click', suppressClickCapture, true)
}

function disarmClickSuppression() {
  window.removeEventListener('click', suppressClickCapture, true)
}

export function TreeDndProvider({
  enabled,
  children,
}: {
  enabled: boolean
  children: React.ReactNode
}) {
  const { t } = useTranslation('viewer')
  const currentPath = useLocation().pathname
  const navigate = useNavigate()
  const [activeNode, setActiveNode] = useState<PageNode | null>(null)
  const [saving, setSaving] = useState(false)
  const subtreeIdsRef = useRef<Set<string>>(new Set())
  const expandTimerRef = useRef<{ nodeId: string; timer: number } | null>(null)
  const disarmClickTimerRef = useRef<number | null>(null)

  const armClickSuppression = useCallback(() => {
    if (disarmClickTimerRef.current) {
      window.clearTimeout(disarmClickTimerRef.current)
      disarmClickTimerRef.current = null
    }
    // 重复注册同一监听器及捕获选项不会增加重复处理函数。
    window.addEventListener('click', suppressClickCapture, true)
  }, [])

  const scheduleClickSuppressionDisarm = useCallback(() => {
    if (disarmClickTimerRef.current) {
      window.clearTimeout(disarmClickTimerRef.current)
    }
    disarmClickTimerRef.current = window.setTimeout(() => {
      disarmClickSuppression()
      disarmClickTimerRef.current = null
    }, 400)
  }, [])

  useEffect(() => disarmClickSuppression, [])

  useEffect(() => {
    useTreeDndStore.setState({ enabled: enabled && !saving })
  }, [enabled, saving])

  const sensors = useSensors(
    // 鼠标移动达到阈值才开始拖动，触摸长按启动拖动并保留普通滚动手势。
    useSensor(MouseSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 250, tolerance: 8 },
    }),
  )

  const clearExpandTimer = () => {
    if (expandTimerRef.current) {
      window.clearTimeout(expandTimerRef.current.timer)
      expandTimerRef.current = null
    }
  }

  const resetDragState = () => {
    // 清除预览、当前拖动与落点，并释放子树缓存和延迟展开计时器。
    setActiveNode(null)
    useTreeDndStore.setState({ activeId: null, dropTarget: null })
    subtreeIdsRef.current = new Set()
    clearExpandTimer()
  }

  const updateDropTarget = (next: DropTarget | null) => {
    const prev = useTreeDndStore.getState().dropTarget
    if (prev?.nodeId === next?.nodeId && prev?.zone === next?.zone) return
    useTreeDndStore.setState({ dropTarget: next })
  }

  const handleDragStart = (event: DragStartEvent) => {
    // 缺少节点数据时不进入拖动状态，成功开始后缓存整棵被拖子树 ID。
    const node = event.active.data.current?.node as PageNode | undefined
    if (!node) return
    armClickSuppression()
    setActiveNode(node)
    useTreeDndStore.setState({ activeId: node.id })
    subtreeIdsRef.current = collectSubtreeIds(node)
  }

  const handleDragMove = (event: DragMoveEvent) => {
    // 没有有效目标、目标属于自身子树或指针区域不可判定时清空落点并取消展开计时。
    const target = event.over?.data.current?.node as PageNode | undefined

    if (!event.over || !target || subtreeIdsRef.current.has(target.id)) {
      updateDropTarget(null)
      clearExpandTimer()
      return
    }

    const pointerY = getPointerY(event.activatorEvent, event.delta)
    const zone =
      pointerY === null ? null : getDropZone(pointerY, event.over.rect)
    if (!zone) {
      updateDropTarget(null)
      clearExpandTimer()
      return
    }

    updateDropTarget({ nodeId: target.id, zone })

    const { isNodeOpen, openNode } = useTreeStore.getState()
    if (
      zone === 'inside' &&
      target.kind === NODE_KIND_SECTION &&
      !isNodeOpen(target.id)
    ) {
      if (expandTimerRef.current?.nodeId !== target.id) {
        // 指针在闭合分区嵌入区域停留满延迟后展开目标；换目标前先取消旧计时器。
        clearExpandTimer()
        const timer = window.setTimeout(() => {
          openNode(target.id)
          expandTimerRef.current = null
        }, EXPAND_ON_HOVER_DELAY_MS)
        expandTimerRef.current = { nodeId: target.id, timer }
      }
    } else {
      clearExpandTimer()
    }
  }

  const performDrop = async (dragged: PageNode, resolution: DropResolution) => {
    // 同父移动只排序；跨父移动先本地展示结果，再提交接口并根据结果刷新索引。
    const { byId, moveNodeLocally, reloadTree, openNode } =
      useTreeStore.getState()
    const currentParentId = dragged.parentId ?? ROOT_ID
    const sameParent = resolution.parentId === currentParentId

    if (sameParent) {
      const siblings = byId[resolution.parentId]?.children ?? []
      const orderedIds = buildOrderedIds(siblings, dragged.id, resolution.index)
      const unchanged = orderedIds.every((id, i) => siblings[i]?.id === id)
      if (unchanged) return

      moveNodeLocally(dragged.id, resolution.parentId, resolution.index)
      setSaving(true)
      try {
        await sortPages(resolution.parentId, orderedIds)
        await reloadTree({ silent: true })
      } catch (err) {
        // 排序失败时重新读取服务端树，撤销临时本地顺序。
        console.warn(err)
        toast.error(t('treeActions.reorderErrorFallback'))
        await reloadTree({ silent: true })
      } finally {
        setSaving(false)
      }
      return
    }

    moveNodeLocally(dragged.id, resolution.parentId, resolution.index)
    if (resolution.parentId !== ROOT_ID) {
      openNode(resolution.parentId)
    }
    setSaving(true)
    try {
      const result = await performTreeCrossParentMove(dragged, resolution)
      // 用户取消预览时刷新服务端树，确保临时的本地跨父移动不遗留。
      if (result.status === 'aborted') {
        await reloadTree({ silent: true })
        return
      }
      if (result.preview) {
        await refreshAfterPageRefactor({
          preview: result.preview,
          currentPath,
          navigate,
          silentReload: true,
        })
      } else {
        await reloadTree({ silent: true })
      }
      if (currentParentId !== ROOT_ID) {
        offerConvertBackToPage(currentParentId)
      }
    } catch (err) {
      console.warn(err)
      toast.error(t('treeActions.moveErrorFallback'))
      await reloadTree({ silent: true })
    } finally {
      setSaving(false)
    }
  }

  const handleDragEnd = (event: DragEndEvent) => {
    // 无有效拖动项、落点或目标节点时结束处理，不提交移动请求。
    scheduleClickSuppressionDisarm()

    const dragged = event.active.data.current?.node as PageNode | undefined
    const target = useTreeDndStore.getState().dropTarget
    resetDragState()

    if (!dragged || !target) return

    const { byId, isNodeOpen } = useTreeStore.getState()
    const targetNode = byId[target.nodeId]
    if (!targetNode) return

    const resolution = resolveDrop({
      dragged,
      target: targetNode,
      zone: target.zone,
      byId,
      isNodeOpen,
    })
    if (!resolution) return

    void performDrop(dragged, resolution)
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={pointerWithin}
      autoScroll={{ threshold: { x: 0, y: 0.2 } }}
      onDragStart={handleDragStart}
      onDragMove={handleDragMove}
      onDragEnd={handleDragEnd}
      onDragCancel={() => {
        scheduleClickSuppressionDisarm()
        resetDragState()
      }}
    >
      {children}
      {/* Portaled to <body>: the sidebar panel animates via a CSS
          transform，若预览留在侧栏内，该变换会成为 fixed 定位参照并造成指针偏移。 */}
      {createPortal(
        <DragOverlay dropAnimation={null} modifiers={[followCursor]}>
          {activeNode ? (
            <div className="tree-dnd__overlay">
              {activeNode.title ||
                i18next.t('treeActions.untitledPage', { ns: 'viewer' })}
            </div>
          ) : null}
        </DragOverlay>,
        document.body,
      )}
    </DndContext>
  )
}
