// Registers a single outline back-arrow toolbar button plus a matching
// 同时注册退出按钮与快捷键动作，并在组件卸载时清理两种入口。

import {
  createHotkeyDefinition,
  getShortcutDisplayLabel,
  type ShortcutId,
} from '@/shared/lib/shortcuts/shortcutCatalog'
import { type HotKeyDefinition, useHotKeysStore } from '@/shared/state/hotkeys'
import { ArrowLeft } from 'lucide-react'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useToolbarStore } from '@/features/toolbar/toolbarStore'

const isMacOS =
  typeof navigator !== 'undefined' &&
  /Mac|iPhone|iPad|iPod/.test(navigator.platform)

export type UseExitModeButtonOptions = {
  id: string
  labelKey: string
  ns: string
  hotkeyId: ShortcutId
  onExit: () => void
}

export function useExitModeButton({
  id,
  labelKey,
  ns,
  hotkeyId,
  onExit,
}: UseExitModeButtonOptions) {
  const { t } = useTranslation(ns)
  const setToolbarButtons = useToolbarStore((state) => state.setButtons)
  const registerHotkey = useHotKeysStore((state) => state.registerHotkey)
  const unregisterHotkey = useHotKeysStore((state) => state.unregisterHotkey)

  useEffect(() => {
    setToolbarButtons([
      {
        id,
        label: t(labelKey),
        hotkey: getShortcutDisplayLabel(hotkeyId, isMacOS),
        icon: <ArrowLeft size={18} />,
        action: onExit,
        variant: 'outline',
      },
    ])

    const hotkey: HotKeyDefinition = createHotkeyDefinition(hotkeyId, onExit)
    registerHotkey(hotkey)

    return () => {
      setToolbarButtons([])
      unregisterHotkey(hotkey)
    }
  }, [
    id,
    labelKey,
    hotkeyId,
    onExit,
    t,
    setToolbarButtons,
    registerHotkey,
    unregisterHotkey,
  ])
}
