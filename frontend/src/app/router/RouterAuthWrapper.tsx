// 使用实时会话与只读状态判断页面入口，包装页面布局及随路径重置的错误边界。
import { ErrorBoundary } from '@/shared/components/ErrorBoundary'
import RequireAuth from '@/features/auth/RequireAuth'
import AppLayout from '@/app/layout/AppLayout'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { useSessionStore } from '@/shared/state/session'
import { Navigate, useLocation } from 'react-router'
import type { ReactNode } from 'react'

// AuthWrapper 仅把无用户且只读的访问视为公开查看，未允许公开的路由跳回首页。
// 其他访问交给 RequireAuth，allowPublic 不直接取消已登录访问的认证包装。
export default function AuthWrapper({
  children,
  allowPublic = false,
}: {
  children: ReactNode
  allowPublic?: boolean
}) {
  const { pathname } = useLocation()
  const user = useSessionStore((state) => state.user)
  const isReadOnly = useIsReadOnly()
  // 公开查看条件同时要求没有缓存用户，普通编辑者只读页面不走匿名分支。
  const publicViewer = isReadOnly && !user
  if (publicViewer && !allowPublic) return <Navigate to="/" replace />
  const content = <AppLayout>{children}</AppLayout>
  return (
    <ErrorBoundary resetKey={pathname}>
      {publicViewer ? content : <RequireAuth>{content}</RequireAuth>}
    </ErrorBoundary>
  )
}
