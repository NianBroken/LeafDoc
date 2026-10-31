// 侧栏在移动布局切换路径时自动关闭，锚点或查询参数变化不计作路径切换。
import { useIsMobile } from '@/shared/lib/useIsMobile'
import { useSidebarStore } from '@/features/sidebar/state/sidebar'
import { useLayoutEffect, useRef } from 'react'
import { useLocation } from 'react-router'

// 上次路径保存在 ref，桌面布局也同步该值；转为移动布局本身不会触发路径差异。
export function useAutoCloseSidebarOnMobile() {
  const location = useLocation()
  const isMobile = useIsMobile()

  const sidebarVisible = useSidebarStore((s) => s.sidebarVisible)
  const setSidebarVisible = useSidebarStore((s) => s.setSidebarVisible)

  const prevPathRef = useRef(location.pathname)

  useLayoutEffect(() => {
    if (!isMobile) {
      prevPathRef.current = location.pathname
      return
    }

    const prevPath = prevPathRef.current
    const nextPath = location.pathname
    prevPathRef.current = nextPath

    // 移动布局下路由路径变化时关闭侧栏。
    if (prevPath !== nextPath && sidebarVisible) {
      setSidebarVisible(false)
    }
  }, [location.pathname, isMobile, sidebarVisible, setSidebarVisible])
}
