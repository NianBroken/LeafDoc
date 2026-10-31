// 此菜单根据会话展示登录按钮或头像入口，并连接设置、快捷键帮助与登出操作。
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from '@/shared/components/ui/avatar'
import { Button } from '@/shared/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/components/ui/dropdown-menu'
import { avatarUrl } from '@/shared/lib/api/avatar'
import * as authAPI from '@/shared/lib/api/auth'
import i18next from '@/shared/lib/i18n'
import { DIALOG_SHORTCUTS_HELP } from '@/shared/lib/registries/instances'
import { useTranslation } from 'react-i18next'
import { redirectToExternal } from '@/shared/lib/redirectToExternal'
import { createHotkeyDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { useAvatarStore } from '@/features/settings/account/state/avatar'
import { useConfigStore } from '@/shared/state/config'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useHotKeysStore } from '@/shared/state/hotkeys'
import { useSessionStore } from '@/shared/state/session'
import { useEffect } from 'react'
import { useNavigate } from 'react-router'
import { RoleGuard } from '@/features/auth/components/RoleGuard'

// 头像地址携带版本以跟随上传变化，菜单权限与页面只读状态分别由共享规则计算。
export default function UserMenu() {
  const { t } = useTranslation('auth')
  const user = useSessionStore((s) => s.user)
  const avatarVersion = useAvatarStore((s) => s.avatarVersion)
  const navigate = useNavigate()
  const openDialog = useDialogsStore((state) => state.openDialog)
  const readOnly = useIsReadOnly()
  const httpRemoteUserEnabled = useConfigStore((s) => s.httpRemoteUserEnabled)
  const registerHotkey = useHotKeysStore((state) => state.registerHotkey)
  const unregisterHotkey = useHotKeysStore((state) => state.unregisterHotkey)
  const logoutUrl = useConfigStore((s) => s.logoutUrl)
  const loginUrl = useConfigStore((s) => s.loginUrl)

  useEffect(() => {
    // 仅已登录且非只读时注册帮助快捷键，依赖变化或卸载时按引用注销。
    if (!user || readOnly) {
      return
    }

    const hotkey = createHotkeyDefinition('shortcuts.help.open', () =>
      openDialog(DIALOG_SHORTCUTS_HELP),
    )

    registerHotkey(hotkey)
    return () => unregisterHotkey(hotkey)
  }, [openDialog, readOnly, registerHotkey, unregisterHotkey, user])

  if (!user) {
    // 外部登录地址存在时离开当前页面，否则使用应用内路由进入登录表单。
    return (
      <div className="user-menu">
        <Button
          size="sm"
          onClick={() =>
            loginUrl ? redirectToExternal(loginUrl) : navigate('/login')
          }
        >
          {t('login.loginButton')}
        </Button>
      </div>
    )
  }

  const handleLogout = async () => {
    if (logoutUrl) {
      // 发起退出请求后立即跳转，不等待请求结束；logout 的 finally 仍会清空本地会话。
      // 此分支不向用户展示退出请求失败，跳转动作与请求结果独立。
      authAPI.logout().catch(() => {})
      redirectToExternal(logoutUrl)
      return
    }
    // 内置退出等待请求完成才导航，请求失败仍由 API 清空会话，但不会执行下一行导航。
    await authAPI.logout()
    navigate('/login')
  }

  return (
    <div className="user-menu">
      <DropdownMenu>
        <DropdownMenuTrigger className="user-menu__dropdown-trigger">
          <Avatar className="user-menu__avatar" data-testid="user-menu-avatar">
            {user && (
              <AvatarImage src={avatarUrl(user.id, avatarVersion)} alt="" />
            )}
            <AvatarFallback className="user-menu__avatar-fallback">
              {user?.username[0].toUpperCase()}
            </AvatarFallback>
          </Avatar>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            className="cursor-pointer"
            onClick={() => navigate('/settings')}
            data-testid="user-menu-settings"
          >
            {t('userMenu.settings')}
          </DropdownMenuItem>
          <RoleGuard roles={['admin', 'editor']}>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="cursor-pointer"
              onClick={() => openDialog(DIALOG_SHORTCUTS_HELP)}
            >
              {i18next.t('shortcutsHelp.menuItem', { ns: 'viewer' })}
            </DropdownMenuItem>
          </RoleGuard>
          {/* 代理认证只有配置外部退出地址才显示退出项，避免本地退出后代理立即重建会话。 */}
          {(!httpRemoteUserEnabled || logoutUrl) && (
            <DropdownMenuItem
              className="cursor-pointer"
              onClick={handleLogout}
              data-testid="user-menu-logout"
            >
              {t('userMenu.logout')}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
