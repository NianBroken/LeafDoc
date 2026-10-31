// 应用装配入口以模块求值副作用向共享注册器登记侧栏和对话框。
// 注册内容引用功能组件，调用方导入本模块即可建立应用级映射。
import {
  dialogRegistry,
  panelItemRegistry,
  SIDEBAR_TREE_PANEL_ID,
  SIDEBAR_SEARCH_PANEL_ID,
  DIALOG_ADD_PAGE,
  DIALOG_SORT_PAGES,
  DIALOG_MOVE_PAGE,
  DIALOG_CREATE_PAGE_BY_PATH,
  DIALOG_COPY_PAGE,
  DIALOG_EDIT_PAGE_METADATA,
  DIALOG_ASSET_MANAGER,
  DIALOG_DELETE_PAGE_CONFIRMATION,
  DIALOG_USER_FORM,
  DIALOG_CHANGE_USER_PASSWORD,
  DIALOG_DELETE_USER_CONFIRMATION,
  DIALOG_API_KEY_FORM,
  DIALOG_DELETE_API_KEY_CONFIRMATION,
  DIALOG_UNSAVED_CHANGES,
  DIALOG_IMAGE_PREVIEW,
  DIALOG_PAGE_QUICK_SWITCHER,
  DIALOG_PAGE_REFACTOR_CONFIRMATION,
  DIALOG_PAGE_PERMALINK,
  DIALOG_RESTORE_REVISION_CONFIRMATION,
  DIALOG_LINK_INSERT,
  DIALOG_WIKILINK_DISAMBIGUATION,
  DIALOG_SHORTCUTS_HELP,
} from '@/shared/lib/registries/instances'
import TreeView from '@/features/tree/TreeView'
import i18next from '@/shared/lib/i18n'
import { getShortcutDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { FolderTree, Search as SearchIcon } from 'lucide-react'
import {
  AddPageDialog,
  ApiKeyFormDialog,
  AssetManagerDialog,
  ChangePasswordDialog,
  CopyPageDialog,
  CreatePageByPathDialog,
  DeleteApiKeyDialog,
  DeletePageDialog,
  DeleteUserDialog,
  EditPageMetadataDialog,
  ImagePreviewDialog,
  LinkInsertDialog,
  MovePageDialog,
  PageQuickSwitcherDialog,
  PageRefactorDialog,
  PermalinkDialog,
  RestoreRevisionDialog,
  Search,
  ShortcutsDialog,
  SortPagesDialog,
  UnsavedChangesDialog,
  UserFormDialog,
  WikiLinkDisambiguationDialog,
} from '@/app/lazy-dialogs'

// 侧栏树和搜索面板共用查看、编辑与历史模式，标签在读取时按当前语言生成。

panelItemRegistry.register({
  id: SIDEBAR_TREE_PANEL_ID,
  label: () => i18next.t('sidebar.explorerTab', { ns: 'common' }),
  hotkey: getShortcutDefinition('sidebar.explorer.open').keyCombo,
  modes: ['view', 'edit', 'history'],
  icon: () => <FolderTree size={16} />,
  render: () => {
    return <TreeView />
  },
})

panelItemRegistry.register({
  id: SIDEBAR_SEARCH_PANEL_ID,
  label: () => i18next.t('sidebar.searchTab', { ns: 'common' }),
  hotkey: getShortcutDefinition('sidebar.search.open').keyCombo,
  modes: ['view', 'edit', 'history'],
  icon: () => <SearchIcon size={16} />,
  render: (props: unknown) => {
    const SearchProps = props as React.ComponentProps<typeof Search>
    return <Search {...SearchProps} />
  },
})

// registerDialog 把组件渲染函数登记到类型名，属性断言不执行运行时校验。
// 默认 key 使用类型名，可选函数按属性生成 key；withoutProps 忽略传入属性。
function registerDialog<P extends object>(
  type: string,
  Dialog: React.JSXElementConstructor<P>,
  options: { key?: (props: P) => string; withoutProps?: boolean } = {},
) {
  dialogRegistry.register({
    type,
    render: (props) => {
      if (options.withoutProps) {
        return <Dialog key={type} {...({} as P)} />
      }
      const typedProps = props as P
      return <Dialog key={options.key?.(typedProps) ?? type} {...typedProps} />
    },
  })
}

// 页面操作对话框把存储属性直接传给各自组件，组件内部负责解释参数。
registerDialog(DIALOG_ADD_PAGE, AddPageDialog)

registerDialog(DIALOG_SORT_PAGES, SortPagesDialog)

registerDialog(DIALOG_MOVE_PAGE, MovePageDialog)

registerDialog(DIALOG_CREATE_PAGE_BY_PATH, CreatePageByPathDialog)

registerDialog(DIALOG_COPY_PAGE, CopyPageDialog)

registerDialog(DIALOG_EDIT_PAGE_METADATA, EditPageMetadataDialog)

registerDialog(DIALOG_ASSET_MANAGER, AssetManagerDialog)

registerDialog(DIALOG_DELETE_PAGE_CONFIRMATION, DeletePageDialog)

registerDialog(DIALOG_USER_FORM, UserFormDialog)

registerDialog(DIALOG_CHANGE_USER_PASSWORD, ChangePasswordDialog)

registerDialog(DIALOG_DELETE_USER_CONFIRMATION, DeleteUserDialog)

// 无需入口属性的表单与全局工具使用 withoutProps，避免透传无关参数。
registerDialog(DIALOG_API_KEY_FORM, ApiKeyFormDialog, { withoutProps: true })

registerDialog(DIALOG_DELETE_API_KEY_CONFIRMATION, DeleteApiKeyDialog)

registerDialog(DIALOG_UNSAVED_CHANGES, UnsavedChangesDialog)

registerDialog(DIALOG_IMAGE_PREVIEW, ImagePreviewDialog)

registerDialog(DIALOG_PAGE_QUICK_SWITCHER, PageQuickSwitcherDialog, {
  withoutProps: true,
})

registerDialog(DIALOG_SHORTCUTS_HELP, ShortcutsDialog, { withoutProps: true })

// 目标页或目标路径不同会产生不同 key，使 React 为对应预览创建独立组件状态。
registerDialog(DIALOG_PAGE_REFACTOR_CONFIRMATION, PageRefactorDialog, {
  key: (props) =>
    `${DIALOG_PAGE_REFACTOR_CONFIRMATION}-${props.preview.pageId}-${props.preview.newPath}`,
})

registerDialog(DIALOG_PAGE_PERMALINK, PermalinkDialog)

registerDialog(DIALOG_LINK_INSERT, LinkInsertDialog)

// 历史恢复对话框按历史 ID 区分组件身份，切换目标历史时不复用旧状态。
registerDialog(DIALOG_RESTORE_REVISION_CONFIRMATION, RestoreRevisionDialog, {
  key: (props) =>
    `${DIALOG_RESTORE_REVISION_CONFIRMATION}-${props.revision.id}`,
})

registerDialog(DIALOG_WIKILINK_DISAMBIGUATION, WikiLinkDisambiguationDialog)
