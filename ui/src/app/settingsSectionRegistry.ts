// The application binds settings metadata to lazy feature components.
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
export const settingsSections = metadata.map((section) => ({
  ...section,
  Component: components[section.id as keyof typeof components],
}))
