import { useEffect, useState } from 'react'
import type { TocEntry } from '@/features/preview/extractTocEntries'

// 监听独立滚动容器，在顶部基准线以下选择最近的目录标题作为当前项。
export function useTocScrollSpy(entries: TocEntry[]): string | null {
  const [activeId, setActiveId] = useState<string | null>(null)

  useEffect(() => {
    if (entries.length === 0) return
    const scrollContainer = document.getElementById('scroll-container')
    if (!scrollContainer) return

    const updateActive = () => {
      // 触发线固定在容器上沿下方 120 像素，活动项按输入目录顺序逐个覆盖。
      const containerRect = scrollContainer.getBoundingClientRect()
      const triggerY = containerRect.top + 120

      // 接近滚动底部时，若短内容使末尾标题无法越过触发线，改选当前视口内的标题。
      const atBottom =
        Math.abs(
          scrollContainer.scrollHeight -
            scrollContainer.scrollTop -
            scrollContainer.clientHeight,
        ) < 5

      let active = entries[0].id
      for (const entry of entries) {
        const el = document.getElementById(entry.id)
        if (!el) continue
        const top = el.getBoundingClientRect().top
        if (top <= triggerY || (atBottom && top < containerRect.bottom)) {
          active = entry.id
        }
      }
      setActiveId(active)
    }

    updateActive()
    scrollContainer.addEventListener('scroll', updateActive, { passive: true })
    return () => scrollContainer.removeEventListener('scroll', updateActive)
  }, [entries])

  return activeId
}
