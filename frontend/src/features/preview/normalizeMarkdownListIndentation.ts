type FenceState = {
  // Markdown 围栏字符用于区分反引号代码块和波浪号代码块。
  marker: '`' | '~'
  // 闭合围栏至少需要与开启围栏相同数量的字符。
  length: number
}

type ListContext = {
  // 输出缩进用于把异常的两格或四格层级规范为四格层级。
  effectiveIndent: number
  // 输入缩进用于识别同级项目、子项目和列表退出位置。
  rawIndent: number
}

type ParsedListLine = {
  containerPrefix: string
  indent: string
  listMarker: string
  rest: string
  spacing: string
}

type ParsedLinePrefix = {
  containerKey: string
  indent: number
}

// 匹配围栏前的引用容器、最多三个空格缩进以及围栏标记。
const fencePattern =
  /^(?<containerPrefix>(?: {0,3}>\s*)*)(?<indent> *)(?<marker>`{3,}|~{3,})(?<rest>.*)$/
// 匹配引用容器内的无序或有序列表项，并保留标记后的原始间距和正文。
const listPattern =
  /^(?<containerPrefix>(?: {0,3}>\s*)*)(?<indent> *)(?<listMarker>(?:[-+*])|(?:\d+[.)]))(?<spacing>\s+)(?<rest>.*)$/
// 提取普通行所属引用层级和缩进，用于判断前一列表上下文是否结束。
const linePrefixPattern = /^(?<containerPrefix>(?: {0,3}>\s*)*)(?<indent> *)/

function getContainerKey(containerPrefix: string) {
  // 列表上下文按引用块深度隔离，避免块内缩进影响块外列表。
  const quoteDepth = (containerPrefix.match(/>/g) ?? []).length
  return `blockquote:${quoteDepth}`
}

function parseListLine(line: string): ParsedListLine | null {
  // 非列表行返回 null，调用方据此清理已结束的列表层级。
  const match = listPattern.exec(line)
  if (!match?.groups) {
    return null
  }

  return {
    containerPrefix: match.groups.containerPrefix ?? '',
    indent: match.groups.indent ?? '',
    listMarker: match.groups.listMarker ?? '',
    spacing: match.groups.spacing ?? ' ',
    rest: match.groups.rest ?? '',
  }
}

function parseLinePrefix(line: string): ParsedLinePrefix {
  // 仅提取行首引用容器和其后的空格，不改动原始行内容。
  const match = linePrefixPattern.exec(line)
  const containerPrefix = match?.groups?.containerPrefix ?? ''

  return {
    containerKey: getContainerKey(containerPrefix),
    indent: (match?.groups?.indent ?? '').length,
  }
}

function getNextFenceState(
  line: string,
  currentFence: FenceState | null,
): FenceState | null {
  // 开围栏进入代码状态，闭围栏要求字符种类相同且长度不短于开围栏。
  const match = fencePattern.exec(line)
  if (!match?.groups) {
    return currentFence
  }

  const markerRun = match.groups.marker
  const marker = markerRun[0] as FenceState['marker']

  if (!currentFence) {
    return {
      marker,
      length: markerRun.length,
    }
  }

  if (
    marker !== currentFence.marker ||
    markerRun.length < currentFence.length
  ) {
    return currentFence
  }

  return null
}

function clearCompletedContexts(
  line: string,
  listContexts: Map<string, ListContext[]>,
) {
  // 空白行不结束列表，普通非空行则按引用深度和缩进清除已退出层级。
  if (line.trim() === '') {
    return
  }

  const parsedPrefix = parseLinePrefix(line)

  for (const [containerKey, stack] of listContexts) {
    if (containerKey !== parsedPrefix.containerKey) {
      listContexts.delete(containerKey)
      continue
    }

    const nextStack = stack.filter(
      (context) => parsedPrefix.indent > context.rawIndent,
    )

    if (nextStack.length === 0) {
      listContexts.delete(containerKey)
      continue
    }

    listContexts.set(containerKey, nextStack)
  }
}

function normalizeListItemIndentation(
  line: string,
  listContexts: Map<string, ListContext[]>,
): string {
  // 为每种引用深度维护独立层级栈，仅调整匹配到的列表项前导空格。
  const parsed = parseListLine(line)
  if (!parsed) {
    clearCompletedContexts(line, listContexts)
    return line
  }

  const rawIndent = parsed.indent.length
  const containerKey = getContainerKey(parsed.containerPrefix)
  const contextStack = [...(listContexts.get(containerKey) ?? [])]

  while (
    contextStack.length > 0 &&
    rawIndent < contextStack[contextStack.length - 1].rawIndent
  ) {
    contextStack.pop()
  }

  let effectiveIndent = rawIndent

  if (contextStack.length > 0) {
    const previousContext = contextStack[contextStack.length - 1]

    if (rawIndent === previousContext.rawIndent) {
      effectiveIndent = previousContext.effectiveIndent
    } else if (rawIndent > previousContext.rawIndent) {
      const indentStep = rawIndent - previousContext.rawIndent
      effectiveIndent =
        previousContext.effectiveIndent +
        (indentStep === 2 || indentStep === 4 ? 4 : indentStep)
    }
  }

  if (
    contextStack.length === 0 ||
    rawIndent !== contextStack[contextStack.length - 1]?.rawIndent
  ) {
    // 相同原始缩进复用现有层级，增加缩进则根据输入步长推导规范层级。
    contextStack.push({
      rawIndent,
      effectiveIndent,
    })
  } else {
    contextStack[contextStack.length - 1] = {
      rawIndent,
      effectiveIndent,
    }
  }

  listContexts.set(containerKey, contextStack)

  if (effectiveIndent === rawIndent) {
    // 原始缩进已经符合当前层级时返回原行，保留其所有字符。
    return line
  }

  return `${parsed.containerPrefix}${' '.repeat(effectiveIndent)}${parsed.listMarker}${parsed.spacing}${parsed.rest}`
}

// 逐行规范列表缩进，代码围栏内部完全跳过以保留示例原文。
export function normalizeMarkdownListIndentation(content: string) {
  const lines = content.split('\n')
  let fenceState: FenceState | null = null
  const listContexts = new Map<string, ListContext[]>()

  return lines
    .map((line) => {
      // 先按当前围栏状态决定是否跳过，再为后续行更新开闭围栏状态。
      const activeFence = fenceState
      fenceState = getNextFenceState(line, fenceState)

      if (activeFence) {
        return line
      }

      return normalizeListItemIndentation(line, listContexts)
    })
    .join('\n')
}
