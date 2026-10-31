// 本文件将页面树展开为搜索项，再按标题、路径和面包屑匹配程度保留有限结果。
import { PageNode } from '@/shared/lib/api/pages'

// 预先保存小写且去掉两端空白的文本，避免每轮查询重复处理同一页面。
type NormalizedFlatPageSearchItem = {
  normalizedTitle: string
  normalizedPath: string
  normalizedBreadcrumb: string
}

// 搜索项同时保存显示原文和匹配文本，不包含页面正文全文。
export type FlatPageSearchItem = {
  id: string
  title: string
  path: string
  kind: PageNode['kind']
  breadcrumb: string
  searchText: string
} & NormalizedFlatPageSearchItem

// 路径前缀分数可由调用方指定，实际使用时只设置上限，不设置下限。
type SearchScoringOptions = {
  pathStartsWithScore?: number
}

// 内部排序条目将计算分数与原项绑定，不把分数写回输入对象。
type ScoredSearchItem = {
  item: FlatPageSearchItem
  score: number
}

// 匹配只忽略大小写和两端空白，不移除重音，也不折叠文本内部空白。
function normalize(value: string) {
  return value.toLowerCase().trim()
}

// 分数高者排前，同分按原始标题的 localeCompare 排列，语言排序遵循运行环境。
function compareScoredItems(a: ScoredSearchItem, b: ScoredSearchItem) {
  if (b.score !== a.score) return b.score - a.score
  return a.item.title.localeCompare(b.item.title)
}

// 将候选插入已排序数组，只保留 limit 项。列表已满且候选不优于末项时直接丢弃。
// 现有同序项先保留，调用方应提供正整数上限。
function pushTopResult(
  results: ScoredSearchItem[],
  entry: ScoredSearchItem,
  limit: number,
) {
  if (
    results.length === limit &&
    compareScoredItems(results[limit - 1], entry) <= 0
  ) {
    return
  }

  let index = 0
  while (
    index < results.length &&
    compareScoredItems(results[index], entry) <= 0
  ) {
    index += 1
  }

  results.splice(index, 0, entry)

  if (results.length > limit) {
    results.pop()
  }
}

// 从根节点的子项开始深度优先展开，不把根本身加入结果。
// path 为空的节点不生成搜索项，但其标题仍进入下级面包屑，子项也继续遍历。
export function buildFlatPageSearchItems(
  root: PageNode | null,
): FlatPageSearchItem[] {
  if (!root) return []

  const items: FlatPageSearchItem[] = []

  // 每次复制祖先标题列表，避免兄弟节点共享可变路径状态。
  const walk = (node: PageNode, parents: string[]) => {
    const breadcrumbParts = [...parents, node.title]

    if (node.path) {
      const breadcrumb = breadcrumbParts.join(' / ')
      const normalizedTitle = normalize(node.title)
      const normalizedPath = normalize(node.path)
      const normalizedBreadcrumb = normalize(breadcrumb)

      items.push({
        id: node.id,
        title: node.title,
        path: node.path,
        kind: node.kind,
        breadcrumb,
        searchText: normalize(
          [node.title, node.path, breadcrumbParts.join(' ')].join(' '),
        ),
        normalizedTitle,
        normalizedPath,
        normalizedBreadcrumb,
      })
    }

    for (const child of node.children || []) {
      walk(child, breadcrumbParts)
    }
  }

  for (const child of root.children || []) {
    walk(child, [])
  }

  return items
}

// 按分支顺序返回首个命中分数，不叠加多个匹配得分。
// 标题优先于路径，之后检查面包屑和多词分散匹配，无命中返回负数。
function scoreItem(
  item: FlatPageSearchItem,
  query: string,
  queryParts: string[],
  options: SearchScoringOptions,
) {
  if (!query) return 0

  const title = item.normalizedTitle
  const path = item.normalizedPath
  const breadcrumb = item.normalizedBreadcrumb

  if (title === query) return 1000
  if (title.startsWith(query)) return 900
  if (title.includes(query)) return 800

  if (path === query) return 700
  if (path.startsWith(query)) {
    if (typeof options.pathStartsWithScore === 'number') {
      return Math.min(options.pathStartsWithScore, 750)
    }

    return 680
  }
  if (breadcrumb.startsWith(query)) return 500
  if (path.includes(query)) return 450
  if (breadcrumb.includes(query)) return 400

  if (
    queryParts.length > 1 &&
    queryParts.every((part) => item.searchText.includes(part))
  ) {
    return 350
  }

  return -1
}

// 空查询保留输入顺序取前 limit 项，非空查询逐项评分并维护有限排序结果。
// limit 不大于零时返回空数组，函数不修改原列表或搜索项。
export function searchFlatPageSearchItems(
  items: FlatPageSearchItem[],
  query: string,
  limit = 20,
  options: SearchScoringOptions = {},
) {
  if (limit <= 0) {
    return []
  }

  const normalizedQuery = normalize(query)

  if (!normalizedQuery) {
    return items.slice(0, limit)
  }

  const queryParts = normalizedQuery.split(/\s+/).filter(Boolean)
  const results: ScoredSearchItem[] = []

  for (const item of items) {
    const score = scoreItem(item, normalizedQuery, queryParts, options)
    if (score < 0) continue

    pushTopResult(results, { item, score }, limit)
  }

  return results.map((entry) => entry.item)
}
