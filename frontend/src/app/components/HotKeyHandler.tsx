// 全局键盘入口按组合键、当前模式、焦点元素和定义条件选择快捷键动作。
// 同组多个定义从后往前寻找首个合格项，每次事件只运行一个动作。

import {
  getHotkeyComboFromEvent,
  isHotkeyAllowedOnElement,
} from '@/shared/lib/hotkeys'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useDialogsStore } from '@/shared/state/dialogs'
import { HotKeyDefinition, useHotKeysStore } from '@/shared/state/hotkeys'
import { useCallback, useEffect } from 'react'

// HotKeyHandler 不渲染界面，挂载期间在 window 捕获阶段监听 keydown。
export function HotKeyHandler() {
  const appMode = useAppMode()
  const isDialogOpen = useDialogsStore((state) => state.isAnyDialogOpen())
  const registeredHotkeys = useHotKeysStore((state) => state.registeredHotkeys)

  // 有对话框时覆盖页面模式，使对话框快捷键在当前事件分发中优先使用独立模式。
  let currentMode = appMode
  if (isDialogOpen) {
    currentMode = 'dialog'
  }

  const onKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // 普通模式跳过已经阻止默认行为的事件，对话框模式仍允许处理这类事件。
      // 这使对话框取消动作能接收其自身关闭处理中已经 preventDefault 的按键。
      if (e.defaultPrevented && currentMode !== 'dialog') {
        return
      }

      const target = e.target instanceof HTMLElement ? e.target : null
      // CodeMirror 搜索区域内的事件全部跳过，包括 Escape。
      if (target?.closest('.cm-search')) {
        return
      }

      const comboString = getHotkeyComboFromEvent(e)

      // 菜单或列表内部保留 Escape 的局部关闭行为，不向页面级动作分发。
      if (
        comboString === 'Escape' &&
        target?.closest('[role="listbox"], [role="menu"]')
      ) {
        return
      }

      // Escape 不受下方表单元素限制，但仍须满足前面的搜索区与菜单过滤。
      if (comboString !== 'Escape') {
        // 按当前焦点元素检查按钮及表单控件，只允许共享规则明确放行的组合键。
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

      // 复制后逆序查找，不改变存储中的注册顺序；禁用或模式不符时继续寻找更早定义。
      const registeredKey = [...registeredKeys]
        .reverse()
        .find(
          (entry) =>
            entry.enabled &&
            entry.mode.includes(currentMode) &&
            (entry.shouldHandle?.() ?? true),
        )

      // 确定有动作后才停止传播并阻止默认行为，随后同步调用该动作。
      if (registeredKey) {
        e.stopPropagation()
        e.preventDefault()
        registeredKey.action()
      }
    },
    [registeredHotkeys, currentMode],
  )

  useEffect(() => {
    // 捕获阶段尽早处理可取消按键，清理使用同一回调与捕获参数移除监听。
    window.addEventListener('keydown', onKeyDown, true)
    return () => {
      window.removeEventListener('keydown', onKeyDown, true)
    }
  }, [onKeyDown])

  return <></>
}
