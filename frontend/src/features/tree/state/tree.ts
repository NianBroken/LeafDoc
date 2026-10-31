// 保存服务端页面树及可供列表查询的双向索引，同时维护展开、选中、收藏固定等界面状态。
// 树刷新后为每个节点补充父 ID，并重建路径索引、ID 索引和扁平搜索数据。
import {
  fetchTree,
  NODE_KIND_PAGE,
  NODE_KIND_SECTION,
  PageNode,
} from '@/shared/lib/api/pages'
import i18next from '@/shared/lib/i18n'
import {
  FlatPageSearchItem,
  buildFlatPageSearchItems,
} from '@/shared/lib/pageSearch'
import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

function buildIndexes(root: PageNode) {
  // 深度遍历根节点及后代，分别按规范路径和 ID 建立引用同一节点对象的索引。
  const byPath: Record<string, PageNode> = {}
  const byId: Record<string, PageNode> = {}

  const walk = (n: PageNode) => {
    byId[n.id] = n
    byPath[n.path] = n
    for (const ch of n.children || []) walk(ch)
  }

  walk(root)
  return { byPath, byId }
}

function collectExpandableNodeIds(root: PageNode | null): string[] {
  // 不包含虚拟根节点，只收集当前确有子项的节点作为可展开项。
  if (!root) return []
  const out: string[] = []

  const walk = (n: PageNode) => {
    const children = n.children || []
    if (children.length > 0) out.push(n.id)
    for (const ch of children) walk(ch)
  }

  for (const ch of root.children || []) {
    walk(ch)
  }
  return out
}

function assignParentIds(node: PageNode, parentId: string | null = null) {
  // 遍历过程中直接写入树节点的 parentId，根节点父级为 null。
  node.parentId = parentId
  for (const child of node.children || []) {
    assignParentIds(child, node.id)
  }
}

function toSetRecord(ids: string[]): Record<string, true> {
  // 将持久化的 ID 数组变为常量时间查询的布尔映射。
  const rec: Record<string, true> = {}
  for (const id of ids) rec[id] = true
  return rec
}

type TreeStore = {
  tree: PageNode | null
  loading: boolean
  error: string | null
  activeNodeId: string | null
  pinnedPages: PageNode[]
  expandAll: () => void
  collapseAll: () => void
  reloadTree: (options?: { silent?: boolean }) => Promise<void>
  patchNodeVersion: (id: string, version: string) => void
  moveNodeLocally: (
    nodeId: string,
    targetParentId: string,
    index: number,
  ) => void
  setPinnedLocally: (id: string, pinned: boolean, version: string) => void
  toggleNode: (id: string) => void
  openNode: (id: string) => void
  closeNode: (id: string) => void
  setActiveNodeId: (id: string | null) => void
  isNodeOpen: (id: string) => boolean
  getPageById: (id: string) => PageNode | null
  getPageByPath: (path: string) => PageNode | null
  getPagesByTitle: (title: string) => PageNode[]
  getPathById: (id: string) => string | null
  getAncestors: (id: string) => string[]
  openAncestorsForPath: (path: string) => void
  openNodeIds: string[]
  openNodeIdSet: Record<string, true>
  byPath: Record<string, PageNode>
  byId: Record<string, PageNode>
  flatPages: FlatPageSearchItem[]
}
export const useTreeStore = create<TreeStore>()(
  persist(
    (set, get) => ({
      tree: null,
      loading: false,
      error: null,
      activeNodeId: null,
      pinnedPages: [],
      openNodeIds: [],
      openNodeIdSet: {},
      byPath: {},
      byId: {},
      flatPages: [],
      expandAll: () => {
        // 根据当前树重新计算有子项的节点 ID，同时替换数组和查询映射。
        const tree = get().tree
        const ids = collectExpandableNodeIds(tree)
        set({ openNodeIds: ids, openNodeIdSet: toSetRecord(ids) })
      },

      collapseAll: () => {
        set({ openNodeIds: [], openNodeIdSet: {} })
      },
      toggleNode: (id: string) => {
        // 通过集合翻转单个节点状态，再同时生成持久化数组和查询映射。
        const current = new Set(get().openNodeIds)

        if (current.has(id)) current.delete(id)
        else current.add(id)

        const ids = Array.from(current)
        set({ openNodeIds: ids, openNodeIdSet: toSetRecord(ids) })
      },

      openNode: (id: string) => {
        // 已打开时不产生新状态引用，未打开时追加 ID 并重建映射。
        if (get().openNodeIdSet?.[id]) {
          return
        }
        const current = new Set(get().openNodeIds)
        current.add(id)
        const ids = Array.from(current)
        set({ openNodeIds: ids, openNodeIdSet: toSetRecord(ids) })
      },

      closeNode: (id: string) => {
        // 已关闭时直接返回，已打开时从数组与映射中一并移除。
        if (!get().openNodeIdSet?.[id]) {
          return
        }
        const current = new Set(get().openNodeIds)
        current.delete(id)
        const ids = Array.from(current)
        set({ openNodeIds: ids, openNodeIdSet: toSetRecord(ids) })
      },

      setActiveNodeId: (id: string | null) => {
        if (get().activeNodeId === id) {
          return
        }
        set({ activeNodeId: id })
      },

      isNodeOpen: (id: string) => !!get().openNodeIdSet?.[id],

      getPageByPath: (path: string) => get().byPath?.[path] ?? null,
      getPageById: (id: string) => get().byId?.[id] ?? null,
      getPagesByTitle: (title: string) => {
        // 只比较小写化后的完整标题，不进行子串匹配或去重。
        const lower = title.toLowerCase()
        return Object.values(get().byId ?? {}).filter(
          (n) => n.title.toLowerCase() === lower,
        )
      },
      getPathById: (id: string) => get().byId?.[id]?.path ?? null,

      getAncestors: (id: string) => {
        // 沿父 ID 向根回溯，并从数组头部插入，返回由近到远的树路径顺序。
        const byId = get().byId
        const out: string[] = []
        let cur = byId?.[id]
        while (cur?.parentId) {
          out.unshift(cur.parentId)
          cur = byId[cur.parentId]
        }
        return out
      },

      openAncestorsForPath: (path: string) => {
        // 路径不存在或目标位于根层时无需修改展开状态。
        const node = get().getPageByPath(path)
        if (!node) return

        const ancestors = get().getAncestors(node.id)
        if (ancestors.length === 0) return

        const merged = new Set(get().openNodeIds)
        let changed = false
        for (const id of ancestors) merged.add(id)
        for (const id of ancestors) {
          // 只在至少一个祖先尚未打开时发布新状态，已完整展开时保持现有引用。
          if (!get().openNodeIdSet?.[id]) {
            changed = true
          }
        }

        if (!changed) {
          return
        }

        const ids = Array.from(merged)
        set({ openNodeIds: ids, openNodeIdSet: toSetRecord(ids) })
      },

      patchNodeVersion: (id: string, version: string) => {
        // 以新对象替换两个索引中的目标节点，不重建树形 children 结构。
        const byId = get().byId
        const byPath = get().byPath
        const node = byId?.[id]
        if (!node) return
        const updatedNode = { ...node, version }
        set({
          byId: { ...byId, [id]: updatedNode },
          byPath: node.path ? { ...byPath, [node.path]: updatedNode } : byPath,
        })
      },

      moveNodeLocally: (
        nodeId: string,
        targetParentId: string,
        index: number,
      ) => {
        // 使用结构化克隆隔离原树，在本地先呈现移动结果，随后由调用方提交并刷新服务端状态。
        const current = get().tree
        if (!current) return

        const tree = structuredClone(current)
        const { byId: clonedById } = buildIndexes(tree)

        const node = clonedById[nodeId]
        const target = clonedById[targetParentId]
        const oldParent = node?.parentId ? clonedById[node.parentId] : null
        if (!node || !target || !oldParent?.children) return

        const oldIndex = oldParent.children.findIndex((c) => c.id === nodeId)
        // 缺少节点、目标或原父子列表时提前退出，不发布部分移动结果。
        if (oldIndex === -1) return
        oldParent.children.splice(oldIndex, 1)

        const children = target.children ?? []
        target.children = children
        const insertAt = Math.max(0, Math.min(index, children.length))
        // 索引夹在插入前子列表的有效边界内，页面作为父节点时同步转成分区。
        children.splice(insertAt, 0, node)
        if (target.kind === NODE_KIND_PAGE) {
          // Moving a node under a page converts that page into a section
          // server-side; mirror it locally so the row updates instantly.
          target.kind = NODE_KIND_SECTION
        }

        assignParentIds(tree)
        // 移动后重新计算父关系、索引、扁平搜索项和按标题排序的固定列表。
        const { byPath, byId } = buildIndexes(tree)
        const flatPages = buildFlatPageSearchItems(tree)
        const pinnedPages = Object.values(byId)
          .filter((n) => n.pinned === true)
          .sort((a, b) => a.title.localeCompare(b.title))
        set({ tree, byPath, byId, flatPages, pinnedPages })
      },

      setPinnedLocally: (id: string, pinned: boolean, version: string) => {
        // 只替换目标节点在索引中的对象，并从新 ID 索引重新派生固定列表。
        const byId = get().byId
        const byPath = get().byPath
        const node = byId?.[id]
        if (!node) return
        const updatedNode = { ...node, pinned, version }
        const updatedById = { ...byId, [id]: updatedNode }
        const pinnedPages = Object.values(updatedById)
          .filter((n) => n.pinned === true)
          .sort((a, b) => a.title.localeCompare(b.title))
        set({
          byId: updatedById,
          byPath: node.path ? { ...byPath, [node.path]: updatedNode } : byPath,
          pinnedPages,
        })
      },

      reloadTree: async (options?: { silent?: boolean }) => {
        // 静默刷新不显示全局加载状态；两种模式都会清除旧错误并请求整棵树。
        const silent = options?.silent === true
        if (silent) set({ error: null })
        else set({ loading: true, error: null })

        try {
          const tree = await fetchTree()
          assignParentIds(tree)
          const { byPath, byId } = buildIndexes(tree)
          const flatPages = buildFlatPageSearchItems(tree)
          const persistedOpen = get().openNodeIds
          // 重载使用请求完成时的展开 ID，建立映射但不丢失用户已保存的展开顺序。
          const pinnedPages = Object.values(byId)
            .filter((n) => n.pinned === true)
            .sort((a, b) => a.title.localeCompare(b.title))
          set({
            tree,
            byPath,
            byId,
            flatPages,
            pinnedPages,
            openNodeIdSet: toSetRecord(persistedOpen),
          })
        } catch (err: unknown) {
          // 错误状态保留旧树数据，仅记录 Error 消息或统一本地化未知错误。
          if (err instanceof Error) {
            set({ error: err.message })
          } else {
            set({
              error: i18next.t('common.unknownError', { ns: 'page' }),
            })
          }
        } finally {
          // 静默模式不写 loading，避免后台刷新改变界面加载指示。
          if (!silent) set({ loading: false })
        }
      },
    }),
    {
      name: 'leafdoc-tree-open-node-ids',
      // 仅将展开节点 ID 写入当前浏览器会话存储，树数据本身每次从服务端读取。
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({
        openNodeIds: state.openNodeIds,
      }),
    },
  ),
)
