// 后代顺序已来自服务端目录，按树遍历顺序使用，不再单独排序。
import { NODE_KIND_PAGE, type PageNode } from '@/shared/lib/api/pages'

// 深度优先检查后代，先遇到的页面直接返回，非页面节点继续检查其 children。
// 起始 section 自身不参与类型判断，整棵后代树没有页面时返回 null。
export function firstDescendantPage(section: PageNode): PageNode | null {
  for (const child of section.children ?? []) {
    if (child.kind === NODE_KIND_PAGE) return child
    const nested = firstDescendantPage(child)
    if (nested) return nested
  }
  return null
}
