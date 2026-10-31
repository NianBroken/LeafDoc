// 把共享设置元数据与应用层组件绑定，供路由装配使用，避免元数据直接依赖功能组件。
import { settingsSections as metadata } from '@/shared/lib/registries/settingsSections'
import {
  AccountSettings,
  PreferencesSettings,
  ApiKeysManagement,
  BrandingSettings,
  Importer,
  PublicAccessSettings,
  SnapshotSettings,
  TocDisplaySettings,
  UserManagement,
  BrokenLinks,
} from '@/app/router/lazy-routes'
// 键须与元数据 id 对应，类型断言不提供运行时缺失项校验。
const components = {
  preferences: PreferencesSettings,
  account: AccountSettings,
  branding: BrandingSettings,
  'public-access': PublicAccessSettings,
  'toc-display': TocDisplaySettings,
  users: UserManagement,
  'api-keys': ApiKeysManagement,
  snapshots: SnapshotSettings,
  importer: Importer,
  brokenLinks: BrokenLinks,
}
// 每个分区复制元数据并附加组件，不在映射过程中实例化或加载懒组件。
export const settingsSections = metadata.map((section) => ({
  ...section,
  Component: components[section.id as keyof typeof components],
}))
