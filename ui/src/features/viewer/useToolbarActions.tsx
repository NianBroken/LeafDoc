// Hook to provide toolbar actions for the page viewer

import { NODE_KIND_PAGE, type Page } from '@/shared/lib/api/pages'
import {
  createHotkeyDefinition,
  getShortcutDisplayLabel,
} from '@/shared/lib/shortcuts/shortcutCatalog'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { useConfigStore } from '@/shared/state/config'
import { HotKeyDefinition, useHotKeysStore } from '@/shared/state/hotkeys'
import { Copy, History, Link2, Pencil, Printer, Trash2 } from 'lucide-react'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  type ToolbarButton,
  useToolbarStore,
} from '@/features/toolbar/toolbarStore'

export interface ToolbarActionsOptions {
  pageKind?: Page['kind']
  printPage: () => void
  editPage: () => void
  showHistory: () => void
  showPermalink: () => void
  deletePage: () => void
  copyPage: () => void
}

export function useToolbarActions({
  pageKind = NODE_KIND_PAGE,
  printPage,
  editPage,
  showHistory,
  showPermalink,
  deletePage,
  copyPage,
}: ToolbarActionsOptions) {
  const { t } = useTranslation('viewer')
  const setButtons = useToolbarStore((state) => state.setButtons)
  const appMode = useAppMode()
  const readOnlyMode = useIsReadOnly()
  const enableRevision = useConfigStore((state) => state.enableRevision)
  const registerHotkey = useHotKeysStore((s) => s.registerHotkey)
  const unregisterHotkey = useHotKeysStore((s) => s.unregisterHotkey)
  const itemLabel =
    pageKind === NODE_KIND_PAGE
      ? t('pageToolbar.itemPage')
      : t('pageToolbar.itemSection')
  const isMacOS =
    typeof navigator !== 'undefined' &&
    /Mac|iPhone|iPad|iPod/.test(navigator.platform)

  useEffect(() => {
    if (readOnlyMode || appMode !== 'view') {
      setButtons([])
      return
    }

    const toolbarButtons: ToolbarButton[] = [
      {
        id: 'edit-page',
        label: t('pageToolbar.editButton', { item: itemLabel }),
        hotkey: getShortcutDisplayLabel('viewer.page.edit', isMacOS),
        icon: <Pencil size={18} />,
        action: editPage,
      },
      {
        id: 'page-permalink',
        label: t('pageToolbar.shareButton', { item: itemLabel }),
        hotkey: getShortcutDisplayLabel('viewer.page.permalink', isMacOS),
        icon: <Link2 size={18} />,
        variant: 'outline',
        action: showPermalink,
      },
      {
        id: 'print-page',
        label: t('pageToolbar.printButton', { item: itemLabel }),
        hotkey: getShortcutDisplayLabel('viewer.page.print', isMacOS),
        icon: <Printer size={18} />,
        action: printPage,
      },
      {
        id: 'copy-page',
        label: t('pageToolbar.copyButton', { item: itemLabel }),
        hotkey: getShortcutDisplayLabel('viewer.page.copy', isMacOS),
        icon: <Copy size={18} />,
        variant: 'outline',
        action: copyPage,
      },
      {
        id: 'delete-page',
        label: t('pageToolbar.deleteButton', { item: itemLabel }),
        hotkey: getShortcutDisplayLabel('viewer.page.delete', isMacOS),
        icon: <Trash2 size={18} />,
        variant: 'outline',
        destructive: true,
        className: 'hover:text-red-600 hover:bg-red-100 hover:border-red-300',
        action: deletePage,
      },
    ]

    if (enableRevision) {
      toolbarButtons.splice(2, 0, {
        id: 'page-history',
        label: t('pageToolbar.historyButton', { item: itemLabel }),
        hotkey: getShortcutDisplayLabel('viewer.page.history', isMacOS),
        icon: <History size={18} />,
        variant: 'outline',
        action: showHistory,
      })
    }

    setButtons(toolbarButtons)

    const copyHotkey: HotKeyDefinition = createHotkeyDefinition(
      'viewer.page.copy',
      copyPage,
    )

    const permalinkHotkey: HotKeyDefinition = createHotkeyDefinition(
      'viewer.page.permalink',
      showPermalink,
    )

    const editHotkey: HotKeyDefinition = createHotkeyDefinition(
      'viewer.page.edit',
      editPage,
    )

    const printHotkey: HotKeyDefinition = createHotkeyDefinition(
      'viewer.page.print',
      printPage,
    )

    const historyHotkey: HotKeyDefinition = createHotkeyDefinition(
      'viewer.page.history',
      showHistory,
    )

    const deleteHotkey: HotKeyDefinition = createHotkeyDefinition(
      'viewer.page.delete',
      deletePage,
    )

    registerHotkey(editHotkey)
    registerHotkey(permalinkHotkey)
    registerHotkey(copyHotkey)
    registerHotkey(printHotkey)
    if (enableRevision) {
      registerHotkey(historyHotkey)
    }
    registerHotkey(deleteHotkey)

    return () => {
      unregisterHotkey(editHotkey)
      unregisterHotkey(permalinkHotkey)
      unregisterHotkey(copyHotkey)
      unregisterHotkey(printHotkey)
      if (enableRevision) {
        unregisterHotkey(historyHotkey)
      }
      unregisterHotkey(deleteHotkey)
    }
  }, [
    appMode,
    readOnlyMode,
    enableRevision,
    setButtons,
    deletePage,
    copyPage,
    editPage,
    printPage,
    showHistory,
    showPermalink,
    registerHotkey,
    unregisterHotkey,
    itemLabel,
    isMacOS,
    t,
  ])
}
