// The HotKeyHandler is responsible for handling hotkey events globally in the application.
// It registers event listeners for keydown events and calls the actions defined in the hotkey map.

import {
  getHotkeyComboFromEvent,
  isHotkeyAllowedOnElement,
} from '@/shared/lib/hotkeys'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useDialogsStore } from '@/shared/state/dialogs'
import { HotKeyDefinition, useHotKeysStore } from '@/shared/state/hotkeys'
import { useCallback, useEffect } from 'react'

export function HotKeyHandler() {
  const appMode = useAppMode()
  const isDialogOpen = useDialogsStore((state) => state.isAnyDialogOpen())
  const registeredHotkeys = useHotKeysStore((state) => state.registeredHotkeys)

  let currentMode = appMode
  if (isDialogOpen) {
    currentMode = 'dialog'
  }

  const onKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Skip events already handled by CodeMirror (e.g. search panel closing),
      // but not in dialog mode: BaseDialog deliberately calls e.preventDefault()
      // in onEscapeKeyDown to prevent Radix from self-closing the dialog, while
      // still relying on HotKeyHandler to dispatch the registered cancelHotkey.
      if (e.defaultPrevented && currentMode !== 'dialog') {
        return
      }

      const target = e.target instanceof HTMLElement ? e.target : null
      if (target?.closest('.cm-search')) {
        return
      }

      const comboString = getHotkeyComboFromEvent(e)

      // Open menus own Escape so closing a list never exits its surrounding page.
      if (
        comboString === 'Escape' &&
        target?.closest('[role="listbox"], [role="menu"]')
      ) {
        return
      }

      // Escape bypasses the button/textarea guard below so it always reaches
      // the registered hotkey (e.g. closing a dialog or the editor).
      if (comboString !== 'Escape') {
        // if the focus is on a button or textarea, we don't trigger hotkeys;
        // this allows normal typing and button interactions
        const activeElement = document.activeElement
        if (
          activeElement &&
          (activeElement.tagName === 'BUTTON' ||
            activeElement.tagName === 'INPUT' ||
            activeElement.tagName === 'SELECT' ||
            activeElement.tagName === 'TEXTAREA')
        ) {
          if (!isHotkeyAllowedOnElement(activeElement, comboString)) {
            return
          }
        }
      }

      const registeredKeys = registeredHotkeys[comboString] as
        HotKeyDefinition[] | undefined

      if (!registeredKeys || registeredKeys.length === 0) {
        return
      }

      // The newest eligible owner handles the current mode, regardless of render order.
      const registeredKey = [...registeredKeys]
        .reverse()
        .find(
          (entry) =>
            entry.enabled &&
            entry.mode.includes(currentMode) &&
            (entry.shouldHandle?.() ?? true),
        )

      if (registeredKey) {
        e.stopPropagation()
        e.preventDefault()
        registeredKey.action()
      }
    },
    [registeredHotkeys, currentMode],
  )

  useEffect(() => {
    // Capture phase so we can preventDefault before the browser handles the
    // same combo (e.g. Ctrl+Alt+P opening print / private-window shortcuts).
    window.addEventListener('keydown', onKeyDown, true)
    return () => {
      window.removeEventListener('keydown', onKeyDown, true)
    }
  }, [onKeyDown])

  return <></>
}
