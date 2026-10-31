// 本模块集中转出管理设置组件，供应用路由用同一动态导入入口选择具体组件。
// 账号与偏好面板不在导出列表中，它们的路由可见性由各自的登记项控制。
// 导出本身不执行权限检查，路由守卫与后端接口分别负责访问控制。
export { default as ApiKeysManagement } from '@/features/apikeys/ApiKeysManagement'
export { default as BrandingSettings } from '@/features/branding/BrandingSettings'
export { default as Importer } from '@/features/importer/Importer'
export { default as PublicAccessSettings } from '@/features/settings/PublicAccessSettings'
export { default as SnapshotSettings } from '@/features/snapshot/SnapshotSettings'
export { default as TocDisplaySettings } from '@/features/settings/TocDisplaySettings'
export { default as UserManagement } from '@/features/users/UserManagement'
export { default as BrokenLinks } from '@/features/links/BrokenLinks'
