// 此路由门禁在没有用户时等待刷新结果，刷新结束后选择内置或外部登录入口。
import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router'
import ExternalRedirect from '@/features/auth/ExternalRedirect'

// children 只在会话存储已有用户时渲染，不在此组件检查具体角色权限。
type Props = {
  children: ReactNode
}

// 记录完整路径、查询和片段，登录后能够恢复原先访问的位置。
export default function RequireAuth({ children }: Props) {
  const user = useSessionStore((state) => state.user)
  const isRefreshing = useSessionStore((state) => state.isRefreshing)
  const loginUrl = useConfigStore((state) => state.loginUrl)
  const location = useLocation()

  if (!user && !isRefreshing) {
    const redirectTo = `${location.pathname}${location.search}${location.hash}`
    if (loginUrl) {
      return <ExternalRedirect to={loginUrl} returnTo={redirectTo} />
    }
    return <Navigate to="/login" replace state={{ redirectTo }} />
  }

  if (!user && isRefreshing) {
    return null // 刷新尚未结束时不显示子页面，也不提前跳转登录。
  }

  return <>{children}</>
}
