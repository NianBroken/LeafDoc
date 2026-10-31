const blockOpenPattern =
  // ::: 类型开启块，最多缩进三个空格，后续非空文本作为折叠块标题。
  /^(?<indent> {0,3}):::\s*(?<type>[A-Za-z][\w-]*)(?:\s+(?<title>\S.*))?\s*$/
const blockClosePattern = /^(?<indent> {0,3}):::\s*$/
// 代码围栏支持反引号和波浪号，并记录缩进、标记字符及连续长度。
const fencedCodePattern = /^(?<indent> {0,3})(?<marker>`{3,}|~{3,})(?<rest>.*)$/

const shoutoutTypeMap: Record<string, string> = {
  caution: 'warning',
  danger: 'error',
  error: 'error',
  fail: 'error',
  failed: 'error',
  failure: 'error',
  info: 'info',
  note: 'info',
  success: 'success',
  tip: 'info',
  warn: 'warning',
  warning: 'warning',
}

function normalizeBlockType(rawType: string) {
  // 将大小写和下划线规范化，再把同义提示类型映射到少量标准类型。
  const normalizedType = rawType.toLowerCase().replace(/_/g, '-')
  return shoutoutTypeMap[normalizedType] ?? normalizedType
}

function prefixQuoteLine(indent: string, line: string) {
  return `${indent}>${line ? ` ${line}` : ''}`
}

type FenceState = {
  markerChar: '`' | '~'
  markerLength: number
}

function getFenceInfo(line: string): FenceState | null {
  const match = line.match(fencedCodePattern)
  if (!match?.groups) {
    return null
  }

  const marker = match.groups.marker ?? ''
  const markerChar = marker[0]
  if (markerChar !== '`' && markerChar !== '~') {
    return null
  }

  return {
    markerChar,
    markerLength: marker.length,
  }
}

function getNextFenceState(line: string, currentFence: FenceState | null) {
  // 当前未处于代码围栏时记录开围栏，处于围栏时仅接受同字符且长度足够的结束标记。
  const nextFence = getFenceInfo(line)

  if (!currentFence) {
    return nextFence
  }

  if (
    nextFence &&
    nextFence.markerChar === currentFence.markerChar &&
    nextFence.markerLength >= currentFence.markerLength
  ) {
    return null
  }

  return currentFence
}

export function normalizeMarkdownBlocks(content: string) {
  // 只转换代码围栏之外结构完整的 ::: 块；畸形或未闭合块原样输出。
  const normalizedContent = content.replace(/\r\n/g, '\n')
  const lines = normalizedContent.split('\n')
  const output: string[] = []
  let outerFence: FenceState | null = null

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]
    const openMatch = line.match(blockOpenPattern)

    if (!openMatch?.groups || outerFence) {
      // 普通行和围栏内行不解释为提示块，同时逐行更新围栏状态。
      output.push(line)
      outerFence = getNextFenceState(line, outerFence)
      continue
    }

    const indent = openMatch.groups.indent ?? ''
    const title = openMatch.groups.title?.trim()
    const blockType = normalizeBlockType(openMatch.groups.type ?? 'info')
    const originalBlockLines = [line]
    const blockLines: string[] = []
    let closingIndex = index + 1
    let innerFence: FenceState | null = null
    let nestedDepth = 1
    let isMalformed = false

    for (; closingIndex < lines.length; closingIndex += 1) {
      // 内层围栏中的 ::: 仅作为代码，嵌套块标记则使外层候选块判为畸形。
      const candidateLine = lines[closingIndex]
      originalBlockLines.push(candidateLine)

      if (!innerFence) {
        if (blockOpenPattern.test(candidateLine)) {
          nestedDepth += 1
          isMalformed = true
          continue
        }

        if (blockClosePattern.test(candidateLine)) {
          nestedDepth -= 1
          if (nestedDepth === 0) {
            break
          }
          continue
        }
      }

      if (!isMalformed) {
        blockLines.push(candidateLine)
      }
      innerFence = getNextFenceState(candidateLine, innerFence)
    }

    if (closingIndex >= lines.length || isMalformed) {
      // 闭合标记缺失或出现嵌套开启标记时保留整个候选区域原文。
      output.push(...originalBlockLines)
      index = closingIndex < lines.length ? closingIndex : lines.length - 1
      continue
    }

    if (output.length > 0 && output[output.length - 1] !== '') {
      output.push('')
    }

    if (blockType === 'collapsible' || blockType === 'collapsed') {
      // 可折叠块转换为 details，其他类型转换为带标准提示标记的引用块。
      appendCollapsibleBlock(output, indent, blockType, blockLines, title)
    } else {
      appendShoutoutBlock(output, indent, blockType, blockLines)
    }

    const nextLine = lines[closingIndex + 1]
    if (nextLine !== undefined && nextLine !== '') {
      output.push('')
    }

    index = closingIndex
  }

  return output.join('\n')
}

function appendCollapsibleBlock(
  output: string[],
  indent: string,
  blockType: 'collapsible' | 'collapsed',
  blockLines: string[],
  title: string | undefined,
) {
  // collapsible 默认展开，collapsed 默认收起；标题作为 summary 文本写入。
  const openAttr = blockType === 'collapsible' ? ' open' : ''

  output.push(`${indent}<details class="markdown-collapsible"${openAttr}>`)

  if (title) {
    output.push(`${indent}<summary>${title}</summary>`)
    output.push('')
  }

  output.push(...blockLines)

  output.push(`${indent}</details>`)
}

function appendShoutoutBlock(
  output: string[],
  indent: string,
  blockType: string,
  blockLines: string[],
) {
  // 标准提示标记作为引用块首行，内容行逐行添加引用前缀。
  output.push(prefixQuoteLine(indent, `[!${blockType.toUpperCase()}]`))
  output.push(prefixQuoteLine(indent, ''))

  if (blockLines.length === 0) {
    output.push(prefixQuoteLine(indent, ''))
    return
  }

  for (const blockLine of blockLines) {
    output.push(prefixQuoteLine(indent, blockLine))
  }
}
