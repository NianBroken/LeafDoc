// 快速切换器沿用共享页面搜索的数据结构与评分规则，只在空查询时保留输入顺序。
import {
  FlatPageSearchItem,
  searchFlatPageSearchItems,
} from '@/shared/lib/pageSearch'

// 此处是同一类型的别名，不复制页面字段，也不加入切换器专属状态。
export type QuickSwitcherItem = FlatPageSearchItem

// 空白查询直接截取前 limit 项；非空查询由共享搜索函数排序并限制结果数。
export function searchQuickSwitcherItems(
  items: QuickSwitcherItem[],
  query: string,
  limit = 20,
) {
  if (!query.trim()) {
    return items.slice(0, limit)
  }

  return searchFlatPageSearchItems(items, query, limit)
}
