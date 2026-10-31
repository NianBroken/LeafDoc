// 本文件按访问标识在当前标签页的 sessionStorage 保存与恢复容器纵向滚动位置。
import { useEffect, useLayoutEffect } from 'react'

// 容器默认标识与存储键前缀共用固定定义，不与浏览器整页滚动位置混用。
const DEFAULT_SCROLL_CONTAINER_ID = 'scroll-container'
const SCROLL_STORAGE_PREFIX = 'scroll:visit:'

// 有地址片段或至少三字符搜索查询时让专用定位逻辑处理，不保存和恢复此访问位置。
// 内容加载完后通过动画帧恢复，卸载或依赖切换时保存当前容器 scrollTop。
// 本 Hook 不捕获存储访问错误，也不取消已经排队的动画帧。
export function useScrollRestoration(
  restorationKey: string,
  isLoading: boolean,
  containerId: string = DEFAULT_SCROLL_CONTAINER_ID,
) {
  useLayoutEffect(() => {
    // 地址片段优先于位置恢复，即使片段目标尚未出现也直接跳过。
    const hash = window.location.hash
    if (hash) return

    // 有效长度的 q 查询由搜索命中滚动处理，短查询仍允许常规恢复。
    const searchQuery = (
      new URLSearchParams(window.location.search).get('q') ?? ''
    ).trim()
    if (searchQuery.length >= 3) return

    if (isLoading) return

    const el = document.getElementById(containerId)
    const stored = sessionStorage.getItem(
      `${SCROLL_STORAGE_PREFIX}${restorationKey}`,
    )

    if (el) {
      // 存储缺失、非法整数或零值都回到顶部，使用 auto 而非平滑滚动。
      requestAnimationFrame(() => {
        el.scrollTo({
          top: stored === null ? 0 : Number.parseInt(stored, 10) || 0,
          behavior: 'auto',
        })
      })
    }
  }, [containerId, isLoading, restorationKey])

  useEffect(() => {
    // 在注册清理时检查片段和查询条件，未通过则不注册本轮位置保存。
    const hash = window.location.hash
    if (hash) return

    const searchQuery = (
      new URLSearchParams(window.location.search).get('q') ?? ''
    ).trim()
    if (searchQuery.length >= 3) return

    return () => {
      const el = document.getElementById(containerId)
      if (el) {
        sessionStorage.setItem(
          `${SCROLL_STORAGE_PREFIX}${restorationKey}`,
          String(el.scrollTop),
        )
      }
    }
  }, [containerId, restorationKey])
}
