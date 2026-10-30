// Descendants already carry the server directory order; never sort a second time.
import { NODE_KIND_PAGE, type PageNode } from '@/shared/lib/api/pages'

export function firstDescendantPage(section: PageNode): PageNode | null {
  for (const child of section.children ?? []) {
    if (child.kind === NODE_KIND_PAGE) return child
    const nested = firstDescendantPage(child)
    if (nested) return nested
  }
  return null
}
