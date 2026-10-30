import { Button } from '@/shared/components/ui/button'
import { DIALOG_PAGE_QUICK_SWITCHER } from '@/shared/lib/registries/instances'
import { createHotkeyDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useHotKeysStore } from '@/shared/state/hotkeys'
import { FileSearch } from 'lucide-react'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

export function PageQuickSwitcherTrigger() {
  const { t } = useTranslation('viewer')
  const appMode = useAppMode()
  const openDialog = useDialogsStore((state) => state.openDialog)
  const registerHotkey = useHotKeysStore((state) => state.registerHotkey)
  const unregisterHotkey = useHotKeysStore((state) => state.unregisterHotkey)

  useEffect(() => {
    const hotkey = createHotkeyDefinition('page.quickSwitcher.open', () =>
      openDialog(DIALOG_PAGE_QUICK_SWITCHER),
    )

    registerHotkey(hotkey)
    return () => unregisterHotkey(hotkey)
  }, [openDialog, registerHotkey, unregisterHotkey])

  if (appMode !== 'view') {
    return null
  }

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={() => openDialog(DIALOG_PAGE_QUICK_SWITCHER)}
      aria-label={t('pageSwitcher.trigger')}
      title={t('pageSwitcher.trigger')}
      className="max-md:px-2"
    >
      <FileSearch size={16} />
      <span className="max-md:hidden">{t('pageSwitcher.trigger')}</span>
    </Button>
  )
}
