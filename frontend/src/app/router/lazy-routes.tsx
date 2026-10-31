// 集中声明路由组件加载入口，React lazy 在组件首次渲染时调用对应导入函数。
import { lazy } from 'react'

export const AccountSettings = lazy(
  () => import('@/features/settings/account/AccountSettings'),
)

// 管理设置共用 adminSections 模块入口，各包装器从同一模块选择自己的命名导出。
// 加载边界只负责代码获取，管理员权限仍由设置守卫判断。
export const ApiKeysManagement = lazy(() =>
  import('@/features/settings/adminSections').then((m) => ({
    default: m.ApiKeysManagement,
  })),
)
export const BrandingSettings = lazy(() =>
  import('@/features/settings/adminSections').then((m) => ({
    default: m.BrandingSettings,
  })),
)
export const Importer = lazy(() =>
  import('@/features/settings/adminSections').then((m) => ({
    default: m.Importer,
  })),
)
export const PublicAccessSettings = lazy(() =>
  import('@/features/settings/adminSections').then((m) => ({
    default: m.PublicAccessSettings,
  })),
)
export const SnapshotSettings = lazy(() =>
  import('@/features/settings/adminSections').then((m) => ({
    default: m.SnapshotSettings,
  })),
)
export const TocDisplaySettings = lazy(() =>
  import('@/features/settings/adminSections').then((m) => ({
    default: m.TocDisplaySettings,
  })),
)
export const UserManagement = lazy(() =>
  import('@/features/settings/adminSections').then((m) => ({
    default: m.UserManagement,
  })),
)
export const BrokenLinks = lazy(() =>
  import('@/features/settings/adminSections').then((m) => ({
    default: m.BrokenLinks,
  })),
)

// 认证和编辑等页面各自使用实际功能模块入口，加载期间由上层 Suspense 显示占位。
export const LoginForm = lazy(() => import('@/features/auth/LoginForm'))
export const ForgotPasswordForm = lazy(
  () => import('@/features/auth/ForgotPasswordForm'),
)
export const ResetPasswordPage = lazy(
  () => import('@/features/auth/ResetPasswordPage'),
)
export const AcceptInvitePage = lazy(
  () => import('@/features/auth/AcceptInvitePage'),
)
export const PageEditor = lazy(() => import('@/features/editor/PageEditor'))
export const PageHistoryPage = lazy(
  () => import('@/features/page/PageHistoryPage'),
)
export const PermalinkRedirect = lazy(
  () => import('@/features/page/PermalinkRedirect'),
)
export const RootRedirect = lazy(() => import('@/features/page/RootRedirect'))
// 查看器使用静态导出，其代码随本模块依赖加载，不经过这里的 lazy 包装。
export { default as PageViewer } from '@/features/viewer/PageViewer'

export const PreferencesSettings = lazy(
  () => import('@/features/settings/PreferencesSettings'),
)
