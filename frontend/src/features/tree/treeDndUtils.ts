// 将树拖放坐标和节点关系解析为父节点 ID 与插入位置，不执行接口调用或修改树状态。
import { NODE_KIND_SECTION, PageNode } from '@/shared/lib/api/pages'

export const ROOT_ID = 'root'

export type DropZone = 'before' | 'after' | 'inside'

export type DropTarget = {
  nodeId: string
  zone: DropZone
}

export type DropResolution = {
  parentId: string
  // 插入索引以移除被拖节点后的目标子列表为基准。
  index: number
}

export function collectSubtreeIds(node: PageNode): Set<string> {
  // 深度遍历拖动节点及全部后代，用于禁止把节点放入自己的子树。
  const ids = new Set<string>()
  const walk = (n: PageNode) => {
    ids.add(n.id)
    for (const child of n.children || []) walk(child)
  }
  walk(node)
  return ids
}

// 节点行按垂直比例分为前四分之一、中央一半和后四分之一，分别对应前置、嵌入和后置。
// 嵌入普通页面时，服务端在移动操作中将目标页转换为分区。
export function getDropZone(
  pointerY: number,
  rect: { top: number; height: number },
): DropZone | null {
  if (rect.height <= 0) return null
  const fraction = (pointerY - rect.top) / rect.height
  // 非正高度没有可计算区域，边界四分之一归入中央嵌入区域。

  if (fraction < 0.25) return 'before'
  if (fraction > 0.75) return 'after'
  return 'inside'
}

export function resolveDrop({
  dragged,
  target,
  zone,
  byId,
  isNodeOpen,
}: {
  dragged: PageNode
  target: PageNode
  zone: DropZone
  byId: Record<string, PageNode>
  isNodeOpen: (id: string) => boolean
}): DropResolution | null {
  // 嵌入总是在目标节点末尾追加；前后投放使用目标的兄弟列表定位。
  if (zone === 'inside') {
    const children = (target.children ?? []).filter((c) => c.id !== dragged.id)
    return { parentId: target.id, index: children.length }
  }

  const parentId = target.parentId ?? ROOT_ID
  const parent = byId[parentId]
  const siblings = (parent?.children ?? []).filter((c) => c.id !== dragged.id)
  const targetIndex = siblings.findIndex((c) => c.id === target.id)
  if (targetIndex === -1) return null

  if (zone === 'after') {
    // 展开的分区后四分之一在界面上对应首个子项上方，存在其他子项时插到子列表开头。
    if (
      target.kind === NODE_KIND_SECTION &&
      isNodeOpen(target.id) &&
      (target.children ?? []).some((c) => c.id !== dragged.id)
    ) {
      return { parentId: target.id, index: 0 }
    }
    return { parentId, index: targetIndex + 1 }
  }

  return { parentId, index: targetIndex }
}

export function buildOrderedIds(
  children: PageNode[],
  draggedId: string,
  index: number,
): string[] {
  // 先剔除旧位置，再把目标 ID 插入裁剪到有效范围内的索引。
  const ids = children.map((c) => c.id).filter((id) => id !== draggedId)
  const insertAt = Math.max(0, Math.min(index, ids.length))
  ids.splice(insertAt, 0, draggedId)
  return ids
}
