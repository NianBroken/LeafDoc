import { createBrowserRouter, Navigate, RouteObject } from 'react-router'
import {
  AcceptInvitePage,
  ForgotPasswordForm,
  LoginForm,
  PageEditor,
  PageHistoryPage,
  PageViewer,
  PermalinkRedirect,
  ResetPasswordPage,
  RootRedirect,
} from '@/app/router/lazy-routes'
import { settingsSections } from '@/app/settingsSectionRegistry'
import ExternalRedirect from '@/features/auth/ExternalRedirect'
import SettingsIndexRedirect from '@/features/settings/SettingsIndexRedirect'
import SettingsLayout from '@/features/settings/SettingsLayout'
import SettingsSectionGuard from '@/features/settings/SettingsSectionGuard'
import AuthWrapper from '@/app/router/RouterAuthWrapper'

export const createLeafDocRouter = (
  enableRevision: boolean,
  loginUrl: string,
  smtpEnabled: boolean,
  basename?: string,
) =>
  createBrowserRouter(
    [
      {
        path: '/login',
        element: loginUrl ? <ExternalRedirect to={loginUrl} /> : <LoginForm />,
      },
      // Password recovery requires SMTP and the built-in login form.
      {
        path: '/forgot-password',
        element:
          loginUrl || !smtpEnabled ? (
            <Navigate to="/login" replace />
          ) : (
            <ForgotPasswordForm />
          ),
      },
      {
        path: '/reset-password',
        element:
          loginUrl || !smtpEnabled ? (
            <Navigate to="/login" replace />
          ) : (
            <ResetPasswordPage />
          ),
      },
      {
        path: '/accept-invite',
        element:
          loginUrl || !smtpEnabled ? (
            <Navigate to="/login" replace />
          ) : (
            <AcceptInvitePage />
          ),
      },
      {
        path: '/',
        element: (
          <AuthWrapper allowPublic>
            <RootRedirect />
          </AuthWrapper>
        ),
      },
      {
        // Compat redirect — user management now lives at /settings/users;
        // SettingsSectionGuard there preserves the external userManagementUrl
        // special case and the admin-role gate.
        path: '/users',
        element: <Navigate to="/settings/users" replace />,
      },
      {
        path: '/settings',
        element: (
          <AuthWrapper>
            <SettingsLayout />
          </AuthWrapper>
        ),
        children: [
          { index: true, element: <SettingsIndexRedirect /> },
          ...settingsSections.map((section) => ({
            path: section.path,
            element: (
              <SettingsSectionGuard section={section}>
                <section.Component />
              </SettingsSectionGuard>
            ),
          })),
        ],
      },
      {
        path: '/e/*',
        element: (
          <AuthWrapper>
            <PageEditor />
          </AuthWrapper>
        ),
      },
      {
        path: '/history/*',
        element: !enableRevision ? (
          <Navigate to="/" replace />
        ) : (
          <AuthWrapper allowPublic>
            <PageHistoryPage />
          </AuthWrapper>
        ),
      },
      {
        path: '/p/:id/:slug?',
        element: (
          <AuthWrapper allowPublic>
            <PermalinkRedirect />
          </AuthWrapper>
        ),
      },
      {
        path: '*',
        element: (
          <AuthWrapper allowPublic>
            <PageViewer />
          </AuthWrapper>
        ),
      },
    ] satisfies RouteObject[],
    { basename: basename || undefined },
  )
