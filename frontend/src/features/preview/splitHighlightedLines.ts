import { Children, ReactNode, cloneElement, isValidElement } from 'react'

/** 将高亮后的 React 子节点按换行拆开，并在拆分文本时保留原元素节点与属性。 */
export function splitHighlightedLines(children: ReactNode): ReactNode[][] {
  // 初始空行用于承接空输入，递归遍历遇到文本换行时再创建后续行。
  const lines: ReactNode[][] = [[]]

  const appendToCurrent = (node: ReactNode) => {
    lines[lines.length - 1].push(node)
  }

  const walk = (node: ReactNode): void => {
    // 空节点和布尔值不产生可见内容，字符串与数字按原字符顺序拆分。
    if (node == null || typeof node === 'boolean') {
      return
    }

    if (typeof node === 'string' || typeof node === 'number') {
      const parts = String(node).split('\n')
      parts.forEach((part, index) => {
        if (index > 0) {
          lines.push([])
        }
        if (part !== '') {
          appendToCurrent(part)
        }
      })
      return
    }

    if (Array.isArray(node) || Children.count(node) > 1) {
      Children.forEach(node, walk)
      return
    }

    if (isValidElement<{ children?: ReactNode }>(node)) {
      // 含换行的文本子节点通过克隆原高亮元素分别包裹各行，其他元素整体保留。
      const child = node.props.children
      if (
        (typeof child === 'string' || typeof child === 'number') &&
        String(child).includes('\n')
      ) {
        const parts = String(child).split('\n')
        parts.forEach((part, index) => {
          if (index > 0) {
            lines.push([])
          }
          if (part !== '') {
            appendToCurrent(
              cloneElement(node, { key: `ln-${lines.length}-${index}` }, part),
            )
          }
        })
        return
      }

      appendToCurrent(node)
      return
    }

    appendToCurrent(node)
  }

  walk(children)

  // 输入以换行结尾时移除产生的空末行，保留中间的空白行。
  if (lines.length > 1 && lines[lines.length - 1].length === 0) {
    lines.pop()
  }

  return lines
}
