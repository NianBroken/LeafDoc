// Registration ownership remains stable when React effects clean up out of order.
import { beforeEach, expect, it, vi } from 'vitest'
import { useHotKeysStore, type HotKeyDefinition } from './hotkeys'

beforeEach(() => useHotKeysStore.setState({ registeredHotkeys: {} }))

it('removes only the definition owned by the unmounting component', () => {
  const dialog: HotKeyDefinition = {
    keyCombo: 'Escape',
    enabled: true,
    mode: ['dialog'],
    action: vi.fn(),
  }
  const editor: HotKeyDefinition = {
    keyCombo: 'Escape',
    enabled: true,
    mode: ['edit'],
    action: vi.fn(),
  }
  const store = useHotKeysStore.getState()
  store.registerHotkey(dialog)
  store.registerHotkey(editor)
  store.unregisterHotkey(dialog)
  expect(store.getRegisteredHotkeys().Escape).toEqual([editor])
  store.unregisterHotkey(editor)
  expect(store.getRegisteredHotkeys()).toEqual({})
})
