import type { Image, Parent, Root, Text } from 'mdast'
import { visit } from 'unist-util-visit'

const IMAGE_WIDTH_PATTERN = /^\s*\{width=([^}]*)\}/
const IMAGE_WIDTH_VALUE_PATTERN = /^(\d+(?:\.\d+)?)%$/

function consumeTextNode(
  parent: Parent,
  index: number,
  textNode: Text,
  length: number,
) {
  // 从紧邻图片的文本节点前部移除尺寸语法，内容耗尽时将该节点从父级数组删除。
  textNode.value = textNode.value.slice(length)

  if (textNode.value.length === 0) {
    parent.children.splice(index, 1)
  }
}

export function remarkImageSize() {
  // 读取图片之后紧邻文本开头的宽度标记，并把通过校验的百分比写入图片属性。
  return (tree: Root) => {
    visit(tree, 'image', (node: Image, index, parent: Parent | undefined) => {
      if (index === undefined || !parent) {
        return
      }

      const nextNode = parent.children[index + 1]

      if (!nextNode || nextNode.type !== 'text') {
        return
      }

      const textNode = nextNode as Text
      const match = textNode.value.match(IMAGE_WIDTH_PATTERN)

      if (!match) {
        return
      }

      consumeTextNode(parent, index + 1, textNode, match[0].length)
      // 语法格式成立后立即消费标记，即使宽度数值随后无效也不会显示该标记文本。

      const widthMatch = match[1].match(IMAGE_WIDTH_VALUE_PATTERN)

      if (!widthMatch) {
        return
      }

      const width = Number(widthMatch[1])

      if (!Number.isFinite(width) || width <= 0 || width > 100) {
        return
      }

      node.data ??= {}
      // 保留插件此前写入的其他 HTML 属性，只覆盖 width。
      node.data.hProperties = {
        ...(node.data.hProperties ?? {}),
        width: `${width}%`,
      }
    })
  }
}
