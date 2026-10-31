import { PDF_EXTENSIONS } from '@/shared/lib/config'
import i18n from '@/shared/lib/i18n'
import { Children, ReactElement, ReactNode, isValidElement } from 'react'
import { defaultUrlTransform, type ExtraProps } from 'react-markdown'

// Markdown 渲染组件共用的节点识别、链接转换和滚动容器查找方法。

export const CLOBBER_PREFIX = ''
export const FOOTNOTE_TARGET_PREFIX = '#user-content-fn'
export const WIKILINK_PROTOCOLS = [
  'wikilink-notfound:',
  'wikilink-ambiguous:',
] as const

export type MarkdownNodeProp = ExtraProps

export type ShoutoutConfig = {
  kind: string
}

function getTextContent(node: ReactNode): string {
  // 递归提取字符串、数字和 React 元素的子内容，忽略其他节点类型。
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node)
  }

  if (Array.isArray(node)) {
    return node.map(getTextContent).join('')
  }

  if (isValidElement<{ children?: ReactNode }>(node)) {
    return getTextContent(node.props.children)
  }

  return ''
}

export function getShoutoutConfig(children: ReactNode): ShoutoutConfig | null {
  // 只读取第一个非空子项，要求它是段落且文本完全匹配标准大写提示标记。
  const childArray = Children.toArray(children)
  const firstChild = childArray.find(
    (child) => typeof child !== 'string' || child.trim() !== '',
  )

  if (!isValidElement<{ children?: ReactNode }>(firstChild)) {
    return null
  }

  if (firstChild.type !== 'p') {
    return null
  }

  const marker = getTextContent(firstChild.props.children).trim()
  const match = marker.match(/^\[!(?<kind>[A-Z][A-Z0-9_-]*)\]$/)
  if (!match?.groups?.kind) {
    return null
  }

  const kind = match.groups.kind.toLowerCase()

  return { kind }
}

export function getSemanticShoutoutTitle(
  kind: string,
  language = i18n.language,
) {
  // 四种内置提示类型返回对应语言的语义名称，自定义类型没有语义标题。
  if (
    kind === 'info' ||
    kind === 'success' ||
    kind === 'warning' ||
    kind === 'error'
  ) {
    return i18n.t(`viewer:semanticAlerts.${kind}`, { lng: language })
  }

  return null
}

export function isPdfSrc(src?: string) {
  // 忽略查询与片段后比较扩展名，不执行 URL 解码或资源探测。
  if (!src) return false
  const withoutQuery = src.split(/[?#]/)[0]
  const ext = withoutQuery.split('.').pop()?.toLowerCase()
  return !!ext && PDF_EXTENSIONS.includes(ext)
}

export function normalizeFootnoteHref(href?: string) {
  // 仅规范指定前缀的脚注链接，其他链接值原样返回。
  if (!href?.startsWith(FOOTNOTE_TARGET_PREFIX)) {
    return href
  }

  return `#${CLOBBER_PREFIX}${href.slice(1)}`
}

export function transformMarkdownUrl(url: string) {
  // 保留两种维基链接内部协议，其他地址交给 react-markdown 默认规则筛选。
  if (WIKILINK_PROTOCOLS.some((protocol) => url.startsWith(protocol))) {
    return url
  }

  return defaultUrlTransform(url)
}

export function isPlainListParagraph(
  child: ReactNode,
): child is ReactElement<{ children?: ReactNode; 'data-line'?: string }> {
  // 仅接受带 children 和可选 data-line 的段落，用于判断可否按普通文本列表项包装。
  if (
    !isValidElement<{ children?: ReactNode; 'data-line'?: string }>(child) ||
    child.type !== 'p'
  ) {
    return false
  }

  const propKeys = Object.keys(child.props)
  return propKeys.every((key) => key === 'children' || key === 'data-line')
}

export function findScrollParent(el: HTMLElement | null): HTMLElement | null {
  // 从元素父节点向外寻找计算样式含 auto 或 scroll 的第一个容器，不检查传入元素本身。
  let node: HTMLElement | null = el?.parentElement ?? null
  while (node) {
    const { overflow, overflowY } = getComputedStyle(node)
    if (/auto|scroll/.test(overflow + overflowY)) return node
    node = node.parentElement
  }
  return null
}
