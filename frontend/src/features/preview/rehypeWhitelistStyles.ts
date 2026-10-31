import type { Root } from 'hast'
import { visit } from 'unist-util-visit'

// 仅允许影响常见排版和尺寸的内联样式属性进入最终 HTML。
const ALLOWED_STYLE_PROPS = new Set([
  'margin',
  'margin-top',
  'margin-bottom',
  'margin-left',
  'margin-right',
  'padding',
  'padding-top',
  'padding-bottom',
  'padding-left',
  'padding-right',
  'text-align',
  'font-weight',
  'font-style',
  'text-decoration',
  'color',
  'background-color',
  'font-size',
  'border',
  'border-width',
  'border-style',
  'border-color',
  'border-radius',
  'width',
  'height',
  'max-width',
  'max-height',
  'min-width',
  'min-height',
  'line-height',
  'letter-spacing',
  'word-spacing',
  'display',
  'vertical-align',
])

function sanitizeStyle(style: string): string | undefined {
  // 按分号拆分声明并去除空项，属性值中的冒号仍会在后续重新拼接。
  const declarations = style
    .split(';')
    .map((d) => d.trim())
    .filter(Boolean)

  const safe: string[] = []

  for (const decl of declarations) {
    // 以第一个冒号前的文本作为属性名，剩余片段合并为属性值。
    const [propRaw, ...rest] = decl.split(':')
    if (!propRaw || rest.length === 0) continue

    const prop = propRaw.trim().toLowerCase()
    const value = rest.join(':').trim()

    // 不在允许列表中的属性直接丢弃。
    if (!ALLOWED_STYLE_PROPS.has(prop)) continue

    // 丢弃包含 url() 或 expression() 的值，避免内联样式加载外部资源或执行表达式。
    if (/(url|expression)\s*\(/i.test(value)) continue

    safe.push(`${prop}: ${value}`)
  }

  if (safe.length === 0) return undefined
  return safe.join('; ')
}

export function rehypeWhitelistStyles() {
  // 遍历 HTML 元素，仅净化字符串形式的 style 属性，其余属性交给后续处理。
  return (tree: Root) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    visit(tree, 'element', (node: any) => {
      // 空样式、缺失样式和非字符串样式不由该插件改写。
      const style = node.properties?.style
      if (!style || typeof style !== 'string') return

      const sanitized = sanitizeStyle(style)
      if (sanitized) {
        // 保留清洗后至少含一项安全声明的样式。
        node.properties.style = sanitized
      } else {
        // 原样式中没有可保留声明时移除整个属性。
        delete node.properties.style
      }
    })
  }
}
