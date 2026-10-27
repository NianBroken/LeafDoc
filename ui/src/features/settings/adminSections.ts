// Barrel for the admin-only settings sections (roles: ['admin']).
// Re-exported from one module so their `lazy()` imports in lazy-routes.tsx
// share a single dynamic-import specifier and land in one chunk instead of
// one per section. Account stays out of this barrel since it's visible to
// non-admin users too, and non-admins shouldn't download admin-only code.
export { default as ApiKeysManagement } from '@/features/apikeys/ApiKeysManagement'
export { default as BrandingSettings } from '@/features/branding/BrandingSettings'
export { default as Importer } from '@/features/importer/Importer'
export { default as PublicAccessSettings } from '@/features/settings/PublicAccessSettings'
export { default as SnapshotSettings } from '@/features/snapshot/SnapshotSettings'
export { default as TocDisplaySettings } from '@/features/settings/TocDisplaySettings'
export { default as UserManagement } from '@/features/users/UserManagement'
export { default as BrokenLinks } from '@/features/links/BrokenLinks'
