// 后代顺序已来自服务端目录，按树遍历顺序使用，不再单独排序。
import { NODE_KIND_PAGE, type PageNode } from '@/shared/lib/api/pages'

export function firstDescendantPage(section: PageNode): PageNode | null {
  for (const child of section.children ?? []) {
    if (child.kind === NODE_KIND_PAGE) return child
    const nested = firstDescendantPage(child)
    if (nested) return nested
  }
  return null
}
