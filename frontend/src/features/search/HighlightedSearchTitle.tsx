// 将搜索返回的精确小写 b 标记转换为加粗节点，其余片段作为文本渲染。
import { Fragment } from 'react'

type HighlightedSearchMarkupProps = {
  text: string
}

const highlightTagPattern = /(<\/?b>)/
// 复用脱离文档的文本域解码实体；无 document 的运行环境直接保留输入文本。
const htmlEntityDecoder =
  typeof document === 'undefined' ? null : document.createElement('textarea')

function decodeHtmlEntities(value: string) {
  if (!value.includes('&') || htmlEntityDecoder == null) {
    return value
  }

  // 读取文本域解析后的 value 作为普通字符串返回，界面不将该字符串作为 HTML 插入。
  // eslint-disable-next-line no-restricted-syntax
  htmlEntityDecoder.innerHTML = value
  return htmlEntityDecoder.value
}

export default function HighlightedSearchTitle({
  text,
}: HighlightedSearchMarkupProps) {
  let insideHighlight = false

  return text.split(highlightTagPattern).map((part, index) => {
    // 分割保留标记本身，开闭标记只切换后续片段状态，不直接产生可见节点。
    if (part === '<b>') {
      insideHighlight = true
      return null
    }

    if (part === '</b>') {
      insideHighlight = false
      return null
    }

    if (part === '') {
      return null
    }

    const text = decodeHtmlEntities(part)
    // 实体解码发生在标记识别之后，实体形式的 b 标记仍作为文本显示。

    if (insideHighlight) {
      return <b key={`title-part-${index}`}>{text}</b>
    }

    return <Fragment key={`title-part-${index}`}>{text}</Fragment>
  })
}
