// 在位置变化后保存非设置页面的位置，设置页面因此可读取进入前的位置。
// 判断只接受 settings 独立路径段，不把 settings-guide 等普通页面当作设置页。

import { useLastWikiLocationStore } from '@/shared/state/lastWikiLocation'
import { stripBasePath } from '@/shared/lib/routePath'
import { useEffect } from 'react'
import { useLocation } from 'react-router'

// isSettingsPath 去除部署基路径后识别设置根路径及其子路径，剥离失败沿用原路径。
function isSettingsPath(pathname: string) {
  const path = stripBasePath(pathname) ?? pathname
  return path === '/settings' || path.startsWith('/settings/')
}

// useTrackLastWikiLocation 保留搜索串和导航 state，不记录 URL hash。
// 只写入共享状态，不在副作用中发起页面跳转。
export function useTrackLastWikiLocation() {
  const location = useLocation()
  const setLocation = useLastWikiLocationStore((s) => s.setLocation)

  useEffect(() => {
    if (isSettingsPath(location.pathname)) return
    setLocation({
      pathname: location.pathname,
      search: location.search,
      state: location.state,
    })
  }, [location.pathname, location.search, location.state, setLocation])
}
