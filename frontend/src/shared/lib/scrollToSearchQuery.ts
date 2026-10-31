// 本文件查找正文中的首个查询命中，插入 mark 并滚动，不建立全文搜索索引。
// 标记类名、正文选择器与滚动容器标识在本文件集中维护。
const HIGHLIGHT_CLASS = 'search-query-highlight'
const CONTENT_SELECTOR = '.page-viewer__content'
const SCROLL_CONTAINER_ID = 'scroll-container'

// 调用方可指定滚动方式、是否等待布局和正文根选择器，默认查找页面查看正文。
export type ScrollToSearchQueryOptions = {
  behavior?: ScrollBehavior
  waitForStableLayout?: boolean
  rootSelector?: string
}

// 清除指定根下本功能的 mark，把子节点搬回原父节点并合并相邻文本节点。
// 不移除其他 mark 类型，也不删除命中文字本身。
export function clearSearchQueryHighlights(root: ParentNode = document): void {
  root.querySelectorAll(`mark.${HIGHLIGHT_CLASS}`).forEach((mark) => {
    const parent = mark.parentNode
    if (!parent) return
    while (mark.firstChild) {
      parent.insertBefore(mark.firstChild, mark)
    }
    parent.removeChild(mark)
    parent.normalize()
  })
}

// 按 DOM 文本节点顺序作小写匹配，跳过脚本、样式、noscript 和纯空白文本。
// 不跨越多个文本节点拼接查找，不检查 CSS 可见性，返回索引为小写文本的字符偏移。
function findFirstTextMatch(
  root: HTMLElement,
  query: string,
): { node: Text; index: number } | null {
  const normalizedQuery = query.toLowerCase()
  if (!normalizedQuery) return null

  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const parent = node.parentElement
      if (!parent) return NodeFilter.FILTER_REJECT
      if (parent.closest('script, style, noscript')) {
        return NodeFilter.FILTER_REJECT
      }
      if (!node.textContent?.trim()) return NodeFilter.FILTER_REJECT
      return NodeFilter.FILTER_ACCEPT
    },
  })

  let current = walker.nextNode()
  while (current) {
    const text = current.textContent ?? ''
    const index = text.toLowerCase().indexOf(normalizedQuery)
    if (index >= 0) {
      return { node: current as Text, index }
    }
    current = walker.nextNode()
  }

  return null
}

// 按给定偏移将原节点拆为前文、命中标记和后文，以 textContent 写入避免解释为 HTML。
// 节点已脱离父级时只返回未挂载的标记，不再次查找或恢复原位置。
function wrapMatch(node: Text, index: number, length: number): HTMLElement {
  const fullText = node.textContent ?? ''
  const before = fullText.slice(0, index)
  const matchText = fullText.slice(index, index + length)
  const after = fullText.slice(index + length)

  const mark = document.createElement('mark')
  mark.className = HIGHLIGHT_CLASS
  mark.textContent = matchText
  mark.setAttribute('data-testid', 'search-query-highlight')

  const parent = node.parentNode
  if (!parent) return mark

  const fragment = document.createDocumentFragment()
  if (before) fragment.appendChild(document.createTextNode(before))
  fragment.appendChild(mark)
  if (after) fragment.appendChild(document.createTextNode(after))
  parent.replaceChild(fragment, node)
  return mark
}

// 按固定间隔观察 scrollHeight，连续稳定或达到最大累计采样时长后执行一次回调。
// 不返回计时器句柄，取消语义由调用方回调中的状态检查实现。
function waitUntilHeightStabilizes(
  element: HTMLElement,
  callback: () => void,
  interval = 250,
  maxTotalTime = 3000,
  stableTime = 500,
) {
  let lastHeight = element.scrollHeight
  let stableFor = 0
  let elapsedTime = 0

  const checkHeight = () => {
    const currentHeight = element.scrollHeight
    if (currentHeight === lastHeight) {
      stableFor += interval
      if (stableFor >= stableTime) {
        callback()
        return
      }
    } else {
      lastHeight = currentHeight
      stableFor = 0
    }
    elapsedTime += interval
    if (elapsedTime < maxTotalTime) {
      setTimeout(checkHeight, interval)
    } else {
      callback()
    }
  }

  setTimeout(checkHeight, interval)
}

/**
 * 查询去掉两端空白后至少三字符才处理，容器不存在时直接返回空操作清理函数。
 * 执行时先清理旧标记，再匹配首个文本节点命中并居中滚动，不修改浏览器地址。
 * 返回的清理函数阻止延迟回调实际工作并移除标记，不清除已经调度的高度采样。
 */
export function scrollToSearchQuery(
  query: string,
  {
    behavior = 'smooth',
    waitForStableLayout = true,
    rootSelector = CONTENT_SELECTOR,
  }: ScrollToSearchQueryOptions = {},
): () => void {
  const trimmed = query.trim()
  if (trimmed.length < 3) return () => {}

  const scrollContainer = document.getElementById(
    SCROLL_CONTAINER_ID,
  ) as HTMLElement | null
  if (!scrollContainer) return () => {}

  let cancelled = false

  const run = () => {
    if (cancelled) return

    const contentRoot = document.querySelector(
      rootSelector,
    ) as HTMLElement | null
    if (!contentRoot) return

    clearSearchQueryHighlights(contentRoot)

    const match = findFirstTextMatch(contentRoot, trimmed)
    if (!match) return

    const mark = wrapMatch(match.node, match.index, trimmed.length)
    mark.scrollIntoView({ behavior, block: 'center', inline: 'nearest' })
  }

  if (waitForStableLayout) {
    waitUntilHeightStabilizes(scrollContainer, run)
  } else {
    run()
  }

  return () => {
    cancelled = true
    const contentRoot = document.querySelector(rootSelector)
    if (contentRoot) clearSearchQueryHighlights(contentRoot)
  }
}
