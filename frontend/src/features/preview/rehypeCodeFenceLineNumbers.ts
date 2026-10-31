import type { Element, Root } from 'hast'
import type { Plugin } from 'unified'
import { visit } from 'unist-util-visit'

function asClassList(
  className: string | number | boolean | (string | number)[] | null | undefined,
): string[] {
  // 将语法树中的类名字符串或数组统一成字符串列表，其他类型视为空列表。
  if (Array.isArray(className)) {
    return className.map(String)
  }
  if (typeof className === 'string') {
    return className.split(/\s+/).filter(Boolean)
  }
  return []
}

/** 识别围栏语言标记末尾的等号，移除标记保留语言类名，并启用代码行号属性。 */
export const rehypeCodeFenceLineNumbers: Plugin<[], Root> = () => {
  return (tree) => {
    visit(tree, 'element', (node: Element) => {
      // 只处理 code 元素；所有语言类名中任一项带末尾等号即启用行号。
      if (node.tagName !== 'code') return

      const classes = asClassList(node.properties?.className)
      let enabled = false

      const nextClasses = classes.map((cls) => {
        if (!cls.startsWith('language-') || !cls.endsWith('=')) {
          return cls
        }
        enabled = true
        const withoutMarker = cls.slice(0, -1)
        return withoutMarker === 'language-'
          ? 'language-plaintext'
          : withoutMarker
      })

      if (!enabled) return

      node.properties = node.properties || {}
      node.properties.className = nextClasses
      node.properties['data-line-numbers'] = 'true'
    })
  }
}
