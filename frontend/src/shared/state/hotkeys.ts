// 按标准化按键组合保存快捷键定义栈，具体键盘事件匹配和模式判断由事件处理组件执行。
// 定义以对象身份注销，避免组件卸载时删除其他组件注册的同键快捷键。

import { normalizeHotkeyCombo } from '@/shared/lib/hotkeys'
import { create } from 'zustand'

// HotKeyDefinition 描述组合键、启用状态、允许模式、动作及可选运行条件。
export type HotKeyDefinition = {
  keyCombo: string
  enabled: boolean
  mode: string[] // 允许处理该快捷键的应用模式，由消费方检查。
  action: () => void
  shouldHandle?: () => boolean
}

type HotKeysStore = {
  registeredHotkeys: Record<string, HotKeyDefinition[]> // 每组按注册次序保存，后注册定义放在数组尾部。
  registerHotkey: (hotKeyDefinition: HotKeyDefinition) => void // 追加原定义对象，不在存储层去重。
  unregisterHotkey: (definition: HotKeyDefinition) => void // 按引用相等移除该对象的全部注册项。
  getRegisteredHotkeys: () => Record<string, HotKeyDefinition[]> // 返回当前映射引用，不创建快照副本。
}

export const useHotKeysStore = create<HotKeysStore>((set, get) => ({
  registeredHotkeys: {},
  registerHotkey: (hotKeyDefinition: HotKeyDefinition) => {
    // 只标准化映射键，保存的定义仍是调用方原对象。
    const normalizedKeyCombo = normalizeHotkeyCombo(hotKeyDefinition.keyCombo)

    set((state) => {
      const existingHotkeys = state.registeredHotkeys[normalizedKeyCombo] || []
      return {
        registeredHotkeys: {
          ...state.registeredHotkeys,
          [normalizedKeyCombo]: [...existingHotkeys, hotKeyDefinition],
        },
      }
    })
  },
  unregisterHotkey: (definition: HotKeyDefinition) => {
    const normalizedKeyCombo = normalizeHotkeyCombo(definition.keyCombo)

    set((state) => {
      const existingHotkeys = state.registeredHotkeys[normalizedKeyCombo] || []
      if (existingHotkeys.length === 0) {
        return state // 没有对应组合时沿用原状态。
      }
      // 组件清理顺序无需与注册顺序一致，只删除同一对象引用，保留其他所有者。
      const updatedHotkeys = existingHotkeys.filter(
        (entry) => entry !== definition,
      )
      const newRegisteredHotkeys = { ...state.registeredHotkeys }
      if (updatedHotkeys.length === 0) {
        delete newRegisteredHotkeys[normalizedKeyCombo] // 空栈不保留映射键。
      } else {
        newRegisteredHotkeys[normalizedKeyCombo] = updatedHotkeys
      }
      return {
        registeredHotkeys: newRegisteredHotkeys,
      }
    })
  },
  getRegisteredHotkeys: () => {
    return get().registeredHotkeys
  },
}))
