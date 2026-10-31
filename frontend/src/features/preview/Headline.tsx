import { Paperclip } from 'lucide-react'
import clsx from 'clsx'
import { createElement, HTMLAttributes, isValidElement, ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

// 递归检查标题子树，避免在已有链接中再嵌套一个锚点链接。
function containsLink(node: ReactNode): boolean {
  if (node == null || typeof node === 'boolean') return false
  if (typeof node === 'string' || typeof node === 'number') return false
  if (Array.isArray(node)) {
    return node.some(containsLink)
  }
  if (!isValidElement(node)) return false
  // React 元素的 props 在运行时承载 href 和子节点，类型在此收窄为所需字段。
  const props = (node.props ?? {}) as { href?: string; children?: ReactNode }

  if (typeof node.type === 'string' && node.type === 'a') {
    return true
  }

  if (typeof props.href === 'string') {
    return true
  }

  return containsLink(props.children)
}

export type HeadlineProps = HTMLAttributes<HTMLHeadingElement> & {
  // 标题级别决定最终创建的 h1 至 h6 标签。
  level: number
  // 标题正文可能包含强调、内链等多个 React 子节点。
  children: ReactNode
  'data-line'?: string
  'data-leafdoc-generated-id'?: string
  node?: unknown
}

// 根据 rehype 生成的标题属性创建语义标题，并为自动生成的锚点提供入口。
export default function Headline({
  level,
  children,
  className,
  id,
  'data-line': dataLine,
  'data-leafdoc-generated-id': hasGeneratedId,
  node,
  ...props
}: HeadlineProps) {
  // node 属于 Markdown 渲染器传入的语法树信息，实际输出使用已映射的 HTML 属性。
  void node
  const { t } = useTranslation('viewer')
  const tagName = `h${level}` as keyof HTMLElementTagNameMap
  // 只有带生成标记且拥有 id 的标题添加复制链接入口。
  const shouldRenderAnchor = hasGeneratedId === 'true' && !!id
  const hasNestedLink = containsLink(children)
  const sectionLinkLabel = t('headline.linkToSection')

  return createElement(
    tagName,
    {
      id,
      className: clsx(className, shouldRenderAnchor && 'anchor'),
      'data-line': dataLine,
      ...props,
    },
    // 标题内部已有链接时只在标题末尾增加图标，避免产生非法的链接嵌套。
    shouldRenderAnchor ? (
      hasNestedLink ? (
        <>
          {children}
          <a
            className="headline-anchor"
            href={`#${id}`}
            aria-label={sectionLinkLabel}
            title={sectionLinkLabel}
          >
            <span className="headline-anchor__icon">
              <Paperclip size={18} />
            </span>
          </a>
        </>
      ) : (
        <a className="headline-anchor headline-anchor--full" href={`#${id}`}>
          {children}
          <span className="headline-anchor__icon">
            <Paperclip size={18} />
          </span>
        </a>
      )
    ) : (
      children
    ),
  )
}
