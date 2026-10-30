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

// Register sidebar panel items here

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

// Register application wide dialogs here using dialogRegistry.register(...)

// registerDialog registers a lazily loaded dialog component under its type.
// The dialog store hands props over untyped; they are passed through as the
// component's own props. key defaults to the dialog type; withoutProps
// renders the dialog without forwarding any props.
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

registerDialog(DIALOG_API_KEY_FORM, ApiKeyFormDialog, { withoutProps: true })

registerDialog(DIALOG_DELETE_API_KEY_CONFIRMATION, DeleteApiKeyDialog)

registerDialog(DIALOG_UNSAVED_CHANGES, UnsavedChangesDialog)

registerDialog(DIALOG_IMAGE_PREVIEW, ImagePreviewDialog)

registerDialog(DIALOG_PAGE_QUICK_SWITCHER, PageQuickSwitcherDialog, {
  withoutProps: true,
})

registerDialog(DIALOG_SHORTCUTS_HELP, ShortcutsDialog, { withoutProps: true })

registerDialog(DIALOG_PAGE_REFACTOR_CONFIRMATION, PageRefactorDialog, {
  key: (props) =>
    `${DIALOG_PAGE_REFACTOR_CONFIRMATION}-${props.preview.pageId}-${props.preview.newPath}`,
})

registerDialog(DIALOG_PAGE_PERMALINK, PermalinkDialog)

registerDialog(DIALOG_LINK_INSERT, LinkInsertDialog)

registerDialog(DIALOG_RESTORE_REVISION_CONFIRMATION, RestoreRevisionDialog, {
  key: (props) =>
    `${DIALOG_RESTORE_REVISION_CONFIRMATION}-${props.revision.id}`,
})

registerDialog(DIALOG_WIKILINK_DISAMBIGUATION, WikiLinkDisambiguationDialog)
