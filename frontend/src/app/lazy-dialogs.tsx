// 集中声明对话框与搜索面板的懒加载组件，注册表只保存包装器，不立即请求模块。
import { lazy } from 'react'

// 命名导出包装为 lazy 所需的 default 字段，实际组件实现仍由各功能模块维护。
export const UnsavedChangesDialog = lazy(() =>
  import('@/features/editor/components/UnsavedChangesDialog').then((m) => ({
    default: m.UnsavedChangesDialog,
  })),
)
export const AssetManagerDialog = lazy(() =>
  import('@/features/assets/AssetManagerDialog').then((m) => ({
    default: m.AssetManagerDialog,
  })),
)
export const WikiLinkDisambiguationDialog = lazy(() =>
  import('@/features/wikilinks/WikiLinkDisambiguationDialog').then((m) => ({
    default: m.WikiLinkDisambiguationDialog,
  })),
)
export const LinkInsertDialog = lazy(() =>
  import('@/features/editor/LinkInsertDialog').then((m) => ({
    default: m.LinkInsertDialog,
  })),
)
export const ImagePreviewDialog = lazy(() =>
  import('@/features/imagepreview/ImagePreviewDialog').then((m) => ({
    default: m.ImagePreviewDialog,
  })),
)
export const RestoreRevisionDialog = lazy(() =>
  import('@/features/history/RestoreRevisionDialog').then((m) => ({
    default: m.RestoreRevisionDialog,
  })),
)
export const PageQuickSwitcherDialog = lazy(() =>
  import('@/features/page-switcher/PageQuickSwitcherDialog').then((m) => ({
    default: m.PageQuickSwitcherDialog,
  })),
)
export const AddPageDialog = lazy(() =>
  import('@/features/page/AddPageDialog').then((m) => ({
    default: m.AddPageDialog,
  })),
)
export const CopyPageDialog = lazy(() =>
  import('@/features/page/CopyPageDialog').then((m) => ({
    default: m.CopyPageDialog,
  })),
)
export const CreatePageByPathDialog = lazy(() =>
  import('@/features/page/CreatePageByPathDialog').then((m) => ({
    default: m.CreatePageByPathDialog,
  })),
)
export const DeletePageDialog = lazy(() =>
  import('@/features/page/DeletePageDialog').then((m) => ({
    default: m.DeletePageDialog,
  })),
)
export const EditPageMetadataDialog = lazy(() =>
  import('@/features/page/EditPageMetadataDialog').then((m) => ({
    default: m.EditPageMetadataDialog,
  })),
)
export const MovePageDialog = lazy(() =>
  import('@/features/page/MovePageDialog').then((m) => ({
    default: m.MovePageDialog,
  })),
)
// 已提供默认导出的模块直接返回 import 结果，无需再次选择命名导出。
export const PermalinkDialog = lazy(
  () => import('@/features/page/PermalinkDialog'),
)
export const PageRefactorDialog = lazy(() =>
  import('@/features/page/PageRefactorDialog').then((m) => ({
    default: m.PageRefactorDialog,
  })),
)
export const SortPagesDialog = lazy(() =>
  import('@/features/page/SortPagesDialog').then((m) => ({
    default: m.SortPagesDialog,
  })),
)
export const ShortcutsDialog = lazy(() =>
  import('@/features/shortcuts/ShortcutsDialog').then((m) => ({
    default: m.ShortcutsDialog,
  })),
)
// 搜索面板与对话框共用此加载入口，但由侧栏注册项决定何时渲染。
export const Search = lazy(() => import('@/features/search/Search'))
export const ChangePasswordDialog = lazy(() =>
  import('@/features/users/ChangePasswordDialog').then((m) => ({
    default: m.ChangePasswordDialog,
  })),
)
export const DeleteUserDialog = lazy(() =>
  import('@/features/users/DeleteUserDialog').then((m) => ({
    default: m.DeleteUserDialog,
  })),
)
export const UserFormDialog = lazy(() =>
  import('@/features/users/UserFormDialog').then((m) => ({
    default: m.UserFormDialog,
  })),
)
export const ApiKeyFormDialog = lazy(() =>
  import('@/features/apikeys/ApiKeyFormDialog').then((m) => ({
    default: m.ApiKeyFormDialog,
  })),
)
export const DeleteApiKeyDialog = lazy(() =>
  import('@/features/apikeys/DeleteApiKeyDialog').then((m) => ({
    default: m.DeleteApiKeyDialog,
  })),
)
