// 根据实例配置创建浏览器路由，访问控制由各路由包装器及服务端共同执行。
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

// createLeafDocRouter 固定认证页面和内容路由，并按设置注册表生成设置子路由。
// basename 为空时不提供部署前缀，调用方负责在配置变化后释放旧路由实例。
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
      // 邮件找回、重置和接受邀请只在启用 SMTP 且未配置外部登录地址时显示。
      // 条件不满足时替换当前历史记录跳到登录页，不留下无效表单入口。
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
        // /users 入口统一跳到用户设置，设置分区守卫负责该页面的实际访问判断。
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
          // 分区元数据提供路径，应用装配结果提供组件，守卫逐个控制访问。
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
        // 历史功能关闭时直接回到首页，开启时允许公开查看模式进入包装器。
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
        // 剩余路径交给页面查看器解析，页面是否存在由内容加载流程判定。
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
