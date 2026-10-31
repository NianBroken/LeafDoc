// 按当前路由文本推断应用显示模式，部署基础路径能够识别时先去掉该前缀。
import { stripBasePath } from '@/shared/lib/routePath'
import { useLocation } from 'react-router'

// 类型包含 dialog 供其他消费者表达模式，但当前 Hook 没有返回 dialog 的分支。
export type AppMode = 'edit' | 'history' | 'view' | 'dialog' | 'settings'

// 编辑要求 /e/ 前缀，历史接受根路径及子路径，设置按 /settings 字符串前缀匹配。
// 其他路径均返回 view，不检查对应路由是否真实存在。
export function useAppMode(): AppMode {
  const location = useLocation()
  const pathname = stripBasePath(location.pathname) ?? location.pathname

  if (pathname.startsWith('/e/')) {
    return 'edit'
  }

  if (
    pathname === '/history' ||
    pathname === '/history/' ||
    pathname.startsWith('/history/')
  ) {
    return 'history'
  }

  if (pathname.startsWith('/settings')) {
    return 'settings'
  }

  return 'view'
}
