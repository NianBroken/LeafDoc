// Settings navigation metadata and visibility rules shared by the router and views.
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

export interface SettingsSectionContext {
  role: string | undefined
  snapshotEnabled: boolean
  enableApiKeyManagement: boolean
  userManagementUrl: string | undefined
  editorLimit: number
}

export interface SettingsSection {
  id: string
  // Path segment nested under /settings, e.g. 'branding' -> /settings/branding.
  path: string
  labelKey: string
  ns: string
  icon: LucideIcon
  // [] = any authenticated user, matching hasRole()'s convention.
  roles: string[]
  isEnabled?: (ctx: SettingsSectionContext) => boolean
  // Only set for sections that link out to an external URL instead of
  // rendering their own route (mirrors today's userManagementUrl case).
  externalHref?: (ctx: SettingsSectionContext) => string | undefined
}

export function isSectionVisible(
  section: SettingsSection,
  ctx: SettingsSectionContext,
): boolean {
  const roleOk = hasRole(ctx.role, section.roles)
  if (!roleOk) return false
  return section.isEnabled ? section.isEnabled(ctx) : true
}

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
    // Always visible to admins; env-managed instances render a status-only
    // view inside the component rather than being hidden.
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
    // At editorLimit === 1, this plan allows only the one owner/admin — no
    // one can ever be added, so the section (and the loophole of adding
    // users this way instead of through the enforced /api/users limit) is
    // hidden entirely rather than shown empty. See ErrEditorLimitReached
    // (internal/core/auth) for the backend enforcement this mirrors.
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
