// 本文件根据地址片段滚动到文档标题，可等待容器高度在连续采样间保持不变。
type ScrollToHeadlineOptions = {
  behavior?: ScrollBehavior
  waitForStableLayout?: boolean
}

// hash 按带 # 的文本处理，目标存在时替换地址片段并滚动，不新增浏览历史项。
// 返回函数只阻止后续滚动，不取消已经排队的高度采样计时器。
export function scrollToHeadlineHash(
  hash: string,
  {
    behavior = 'smooth',
    waitForStableLayout = true,
  }: ScrollToHeadlineOptions = {},
): () => void {
  const contentContainer = document.getElementById(
    'scroll-container',
  ) as HTMLElement | null
  if (!contentContainer) return () => {}

  let cancelled = false

  // 默认每 250 毫秒比较一次高度，连续 500 毫秒相同或累计到 3000 毫秒后执行。
  // 这里按采样次数累计时长，不测量真实时钟，不判断图片或其他异步资源是否全部完成。
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

  // 到执行时才解析片段和查找标题，允许等待期间文档继续渲染。
  const scrollToTarget = () => {
    if (cancelled) return

    const rawHeadlineId = hash.substring(1)
    let headlineId = rawHeadlineId

    // 片段编码不合法时按原始 ID 查找，缺失目标会输出原有控制台告警。
    try {
      headlineId = decodeURIComponent(rawHeadlineId)
    } catch {
      headlineId = rawHeadlineId
    }

    const headlineElement = document.getElementById(headlineId)
    if (!headlineElement) {
      console.warn(`Headline with id "${headlineId}" not found.`)
      return
    }
    // 替换地址时传入 null 状态，不保留原 history.state 对象。
    history.replaceState(null, '', hash)
    headlineElement.scrollIntoView({ behavior, block: 'start' })
  }

  if (waitForStableLayout) {
    waitUntilHeightStabilizes(contentContainer, scrollToTarget)
  } else {
    scrollToTarget()
  }

  return () => {
    cancelled = true
  }
}
