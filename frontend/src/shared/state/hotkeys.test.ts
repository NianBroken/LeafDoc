// React 效果清理顺序变化时，快捷键注册仍由当前拥有者稳定维护。
import { beforeEach, expect, it, vi } from 'vitest'
import { useHotKeysStore, type HotKeyDefinition } from './hotkeys'

// 每例清空注册映射，保留存储原有方法，使注册与注销都经过生产实现。
beforeEach(() => useHotKeysStore.setState({ registeredHotkeys: {} }))

it('removes only the definition owned by the unmounting component', () => {
  // 两个定义使用同一组合键而对象身份不同，模式与动作只作为随定义保存的数据。
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
  // 先注销较早登记的对话框，编辑器定义仍须留在栈内，不能直接删除整组按键。
  store.unregisterHotkey(dialog)
  expect(store.getRegisteredHotkeys().Escape).toEqual([editor])
  // 最后一项注销后映射键也应消失，不保留空数组。
  store.unregisterHotkey(editor)
  expect(store.getRegisteredHotkeys()).toEqual({})
})
