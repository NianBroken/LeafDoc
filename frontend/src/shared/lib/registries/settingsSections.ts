// 设置导航与路由共用分区元数据和可见性规则，实际组件由应用层绑定。
import { hasRole } from '@/shared/lib/roles'

import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import {
  Camera,
  Globe,
  KeyRound,
  ListTree,
  Palette,
  Upload,
  User,
  Users,
  Link2Off,
  type LucideIcon,
} from 'lucide-react'

// SettingsSectionContext 从当前会话与实例配置收集可见性判断需要的字段。
export interface SettingsSectionContext {
  role: string | undefined
  snapshotEnabled: boolean
  enableApiKeyManagement: boolean
  userManagementUrl: string | undefined
  editorLimit: number
}

// SettingsSection 定义分区身份、嵌套路由、翻译键、图标及可选启用和外部地址规则。
export interface SettingsSection {
  id: string
  // settings 下的相对路径段，例如 branding 对应 /settings/branding。
  path: string
  labelKey: string
  ns: string
  icon: LucideIcon
  // 空数组采用 hasRole 的规则，不额外限定角色，认证入口由上层包装器约束。
  roles: string[]
  isEnabled?: (ctx: SettingsSectionContext) => boolean
  // 提供外部管理地址的读取函数，由消费方决定跳转和显示方式。
  externalHref?: (ctx: SettingsSectionContext) => string | undefined
}

// isSectionVisible 先匹配角色，再检查可选启用规则，不检查 externalHref 是否存在。
export function isSectionVisible(
  section: SettingsSection,
  ctx: SettingsSectionContext,
): boolean {
  const roleOk = hasRole(ctx.role, section.roles)
  if (!roleOk) return false
  return section.isEnabled ? section.isEnabled(ctx) : true
}

// 数组顺序也是消费方默认使用的导航次序，分区 ID 与路由路径分别保存。
export const settingsSections: SettingsSection[] = [
  {
    id: 'preferences',
    path: 'preferences',
    labelKey: 'nav.preferences',
    ns: 'settings',
    icon: Globe,
    roles: ['admin'],
  },
  {
    id: 'account',
    path: 'account',
    labelKey: 'nav.account',
    ns: 'settings',
    icon: User,
    roles: [],
  },
  {
    id: 'branding',
    path: 'branding',
    labelKey: 'userMenu.brandingSettings',
    ns: 'auth',
    icon: Palette,
    roles: ['admin'],
  },
  {
    id: 'public-access',
    path: 'public-access',
    labelKey: 'menuLabel',
    ns: 'publicAccess',
    icon: Globe,
    roles: ['admin'],
    // 此分区没有额外启用条件，管理员可见，环境管理状态由页面组件显示。
  },
  {
    id: 'toc-display',
    path: 'toc-display',
    labelKey: 'menuLabel',
    ns: 'tocDisplay',
    icon: ListTree,
    roles: ['admin'],
  },
  {
    id: 'users',
    path: 'users',
    labelKey: 'userMenu.userManagement',
    ns: 'auth',
    icon: Users,
    roles: ['admin'],
    // 配额精确等于一时隐藏整个用户分区，这是前端可见性规则。
    // 后端编辑者配额只计管理员与编辑者，不据此断言所有角色都不能创建。
    isEnabled: (ctx) => ctx.editorLimit !== 1,
    externalHref: (ctx) => ctx.userManagementUrl,
  },
  {
    id: 'api-keys',
    path: 'api-keys',
    labelKey: 'menu.title',
    ns: 'apikeys',
    icon: KeyRound,
    roles: ['admin'],
    isEnabled: (ctx) => ctx.enableApiKeyManagement,
  },
  {
    id: 'snapshots',
    path: 'snapshots',
    labelKey: 'menuLabel',
    ns: 'snapshot',
    icon: Camera,
    roles: ['admin'],
    isEnabled: (ctx) => ctx.snapshotEnabled,
  },
  {
    id: 'importer',
    path: 'importer',
    labelKey: 'userMenu.import',
    ns: 'auth',
    icon: Upload,
    roles: ['admin'],
  },
  {
    id: 'brokenLinks',
    path: 'brokenLinks',
    labelKey: 'menuLabel',
    ns: 'brokenLinks',
    icon: Link2Off,
    roles: ['admin'],
  },
]

// useSettingsSectionContext 订阅实时角色与配置，把空外部地址统一表示为 undefined。
// 返回对象每次执行均重新创建，读取本身不发起网络请求。
export function useSettingsSectionContext(): SettingsSectionContext {
  const role = useSessionStore((s) => s.user?.role)
  const snapshotEnabled = useConfigStore((s) => s.snapshotEnabled)
  const enableApiKeyManagement = useConfigStore((s) => s.enableApiKeyManagement)
  const userManagementUrl = useConfigStore((s) => s.userManagementUrl)
  const editorLimit = useConfigStore((s) => s.editorLimit)

  return {
    role,
    snapshotEnabled,
    enableApiKeyManagement,
    userManagementUrl: userManagementUrl || undefined,
    editorLimit,
  }
}
