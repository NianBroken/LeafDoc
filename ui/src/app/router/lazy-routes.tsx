import { lazy } from 'react'

export const AccountSettings = lazy(
  () => import('@/features/settings/account/AccountSettings'),
)

// The admin-only settings sections share one dynamic-import specifier
// (../settings/adminSections) so the bundler collapses them into a single
// chunk, loaded once when an admin opens any one of them. One chunk per
// section saved almost no bytes but cost a network round-trip on every
// switch between settings tabs.
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
export { default as PageViewer } from '@/features/viewer/PageViewer'

export const PreferencesSettings = lazy(
  () => import('@/features/settings/PreferencesSettings'),
)
