// 集中保存快捷键的身份、组合、显示文字和适用模式，供帮助列表与注册入口共同读取。
// 此文件只提供定义与筛选，不监听键盘事件，也不把 customizable 标记保存为用户配置。
import { AppMode } from '@/shared/lib/useAppMode'
import { HotKeyDefinition } from '@/shared/state/hotkeys'

// 各业务调用使用稳定身份查找定义，组合键变化无需修改调用处的身份字符串。
export type ShortcutId =
  | 'dialog.close'
  | 'dialog.confirm'
  | 'shortcuts.help.open'
  | 'page.quickSwitcher.open'
  | 'sidebar.explorer.open'
  | 'sidebar.search.open'
  | 'viewer.page.edit'
  | 'viewer.page.permalink'
  | 'viewer.page.print'
  | 'viewer.page.copy'
  | 'viewer.page.delete'
  | 'viewer.page.history'
  | 'viewer.toc.toggle'
  | 'editor.page.close'
  | 'editor.page.save'
  | 'editor.format.bold'
  | 'editor.format.italic'
  | 'editor.heading.one'
  | 'editor.heading.two'
  | 'editor.heading.three'
  | 'editor.format.inlineCode'
  | 'editor.link.insert'
  | 'history.page.close'
  | 'settings.exit'
  | 'asset.rename.confirm'
  | 'asset.rename.cancel'

// 显示文字与实际匹配组合分别保存，Mac 标签只用于呈现，不改变 Mod 的匹配规则。
export type ShortcutDefinition = {
  id: ShortcutId
  labelKey: string
  categoryKey: string
  keyCombo: string
  defaultDisplayLabel: string
  macDisplayLabel?: string
  // 数组同时供模式筛选与处理器注册使用，帮助列表按数组长度和翻译键排序。
  modes: AppMode[]
  customizable: boolean
  // 仅显式 false 隐藏帮助条目，未提供此字段时仍可显示。
  showInShortcutsDialog?: boolean
}

// 对话框、跨页面导航和各工作模式共享同一清单，相同组合可在不同模式表示不同动作。
export const shortcutDefinitions: ShortcutDefinition[] = [
  {
    id: 'dialog.close',
    labelKey: 'shortcutsHelp.items.closeDialog.action',
    categoryKey: 'shortcutsHelp.categories.dialogs',
    keyCombo: 'Escape',
    defaultDisplayLabel: 'Esc',
    modes: ['dialog'],
    customizable: false,
  },
  {
    id: 'dialog.confirm',
    labelKey: 'shortcutsHelp.items.confirmDialog.action',
    categoryKey: 'shortcutsHelp.categories.dialogs',
    keyCombo: 'Enter',
    defaultDisplayLabel: 'Enter',
    modes: ['dialog'],
    customizable: false,
  },
  {
    id: 'shortcuts.help.open',
    // 帮助入口跨查看、编辑、历史和设置模式显示，匹配物理斜线键。
    labelKey: 'shortcutsHelp.items.openShortcutsHelp.action',
    categoryKey: 'shortcutsHelp.categories.navigation',
    keyCombo: 'Mod+Slash',
    defaultDisplayLabel: 'Ctrl+/',
    macDisplayLabel: 'Cmd+/',
    modes: ['view', 'edit', 'history', 'settings'],
    customizable: true,
  },
  {
    id: 'page.quickSwitcher.open',
    labelKey: 'shortcutsHelp.items.goToPage.action',
    categoryKey: 'shortcutsHelp.categories.navigation',
    keyCombo: 'Mod+Alt+KeyP',
    defaultDisplayLabel: 'Ctrl+Alt+P',
    macDisplayLabel: 'Cmd+Option+P',
    modes: ['view'],
    customizable: true,
  },
  {
    id: 'sidebar.explorer.open',
    // 侧栏快捷键只声明包含侧栏的三个模式，不包含设置模式。
    labelKey: 'shortcutsHelp.items.openExplorer.action',
    categoryKey: 'shortcutsHelp.categories.navigation',
    keyCombo: 'Mod+Shift+KeyE',
    defaultDisplayLabel: 'Ctrl+Shift+E',
    macDisplayLabel: 'Cmd+Shift+E',
    modes: ['view', 'edit', 'history'],
    customizable: true,
  },
  {
    id: 'sidebar.search.open',
    labelKey: 'shortcutsHelp.items.openSearch.action',
    categoryKey: 'shortcutsHelp.categories.navigation',
    keyCombo: 'Mod+Shift+KeyF',
    defaultDisplayLabel: 'Ctrl+Shift+F',
    macDisplayLabel: 'Cmd+Shift+F',
    modes: ['view', 'edit', 'history'],
    customizable: true,
  },
  {
    id: 'viewer.page.edit',
    // 查看模式动作在此只声明组合和模式，是否可执行由注册处的条件处理。
    labelKey: 'shortcutsHelp.items.editPage.action',
    categoryKey: 'shortcutsHelp.categories.viewing',
    keyCombo: 'Mod+KeyE',
    defaultDisplayLabel: 'Ctrl+E',
    macDisplayLabel: 'Cmd+E',
    modes: ['view'],
    customizable: true,
  },
  {
    id: 'viewer.page.permalink',
    labelKey: 'shortcutsHelp.items.sharePage.action',
    categoryKey: 'shortcutsHelp.categories.viewing',
    keyCombo: 'Mod+Shift+KeyL',
    defaultDisplayLabel: 'Ctrl+Shift+L',
    macDisplayLabel: 'Cmd+Shift+L',
    modes: ['view'],
    customizable: true,
  },
  {
    id: 'viewer.page.print',
    labelKey: 'shortcutsHelp.items.printPage.action',
    categoryKey: 'shortcutsHelp.categories.viewing',
    keyCombo: 'Mod+KeyP',
    defaultDisplayLabel: 'Ctrl+P',
    macDisplayLabel: 'Cmd+P',
    modes: ['view'],
    customizable: true,
  },
  {
    id: 'viewer.page.copy',
    labelKey: 'shortcutsHelp.items.copyPage.action',
    categoryKey: 'shortcutsHelp.categories.viewing',
    keyCombo: 'Mod+Shift+KeyS',
    defaultDisplayLabel: 'Ctrl+Shift+S',
    macDisplayLabel: 'Cmd+Shift+S',
    modes: ['view'],
    customizable: true,
  },
  {
    id: 'viewer.page.delete',
    labelKey: 'shortcutsHelp.items.deletePage.action',
    categoryKey: 'shortcutsHelp.categories.viewing',
    keyCombo: 'Mod+Delete',
    defaultDisplayLabel: 'Ctrl+Delete',
    macDisplayLabel: 'Cmd+Delete',
    modes: ['view'],
    customizable: true,
  },
  {
    id: 'viewer.page.history',
    labelKey: 'shortcutsHelp.items.pageHistory.action',
    categoryKey: 'shortcutsHelp.categories.viewing',
    keyCombo: 'Mod+KeyH',
    defaultDisplayLabel: 'Ctrl+H',
    macDisplayLabel: 'Cmd+H',
    modes: ['view'],
    customizable: true,
  },
  {
    id: 'viewer.toc.toggle',
    labelKey: 'shortcutsHelp.items.toggleToc.action',
    categoryKey: 'shortcutsHelp.categories.viewing',
    keyCombo: 'Mod+Shift+KeyO',
    defaultDisplayLabel: 'Ctrl+Shift+O',
    macDisplayLabel: 'Cmd+Shift+O',
    modes: ['view'],
    customizable: true,
  },
  {
    id: 'editor.page.close',
    // Escape 在编辑模式表示关闭编辑器，与对话框和历史模式各自的 Escape 定义独立。
    labelKey: 'shortcutsHelp.items.closeEditor.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Escape',
    defaultDisplayLabel: 'Esc',
    modes: ['edit'],
    customizable: false,
  },
  {
    id: 'editor.page.save',
    labelKey: 'shortcutsHelp.items.savePage.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Mod+KeyS',
    defaultDisplayLabel: 'Ctrl+S',
    macDisplayLabel: 'Cmd+S',
    modes: ['edit'],
    customizable: true,
  },
  {
    id: 'editor.format.bold',
    labelKey: 'shortcutsHelp.items.formatBold.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Mod+KeyB',
    defaultDisplayLabel: 'Ctrl+B',
    macDisplayLabel: 'Cmd+B',
    modes: ['edit'],
    customizable: true,
  },
  {
    id: 'editor.format.italic',
    labelKey: 'shortcutsHelp.items.formatItalic.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Mod+KeyI',
    defaultDisplayLabel: 'Ctrl+I',
    macDisplayLabel: 'Cmd+I',
    modes: ['edit'],
    customizable: true,
  },
  {
    id: 'editor.heading.one',
    // 标题级别按数字物理键区分，显示文字与键盘事件使用的 Digit 名称分开保存。
    labelKey: 'shortcutsHelp.items.insertHeadingOne.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Mod+Alt+Digit1',
    defaultDisplayLabel: 'Ctrl+Alt+1',
    macDisplayLabel: 'Cmd+Option+1',
    modes: ['edit'],
    customizable: true,
  },
  {
    id: 'editor.heading.two',
    labelKey: 'shortcutsHelp.items.insertHeadingTwo.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Mod+Alt+Digit2',
    defaultDisplayLabel: 'Ctrl+Alt+2',
    macDisplayLabel: 'Cmd+Option+2',
    modes: ['edit'],
    customizable: true,
  },
  {
    id: 'editor.heading.three',
    labelKey: 'shortcutsHelp.items.insertHeadingThree.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Mod+Alt+Digit3',
    defaultDisplayLabel: 'Ctrl+Alt+3',
    macDisplayLabel: 'Cmd+Option+3',
    modes: ['edit'],
    customizable: true,
  },
  {
    id: 'editor.format.inlineCode',
    labelKey: 'shortcutsHelp.items.formatInlineCode.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Mod+Backquote',
    defaultDisplayLabel: 'Ctrl+`',
    macDisplayLabel: 'Cmd+`',
    modes: ['edit'],
    customizable: true,
  },
  {
    id: 'editor.link.insert',
    labelKey: 'shortcutsHelp.items.insertLink.action',
    categoryKey: 'shortcutsHelp.categories.editing',
    keyCombo: 'Mod+KeyK',
    defaultDisplayLabel: 'Ctrl+K',
    macDisplayLabel: 'Cmd+K',
    modes: ['edit'],
    customizable: true,
  },
  {
    id: 'history.page.close',
    labelKey: 'shortcutsHelp.items.backToPage.action',
    categoryKey: 'shortcutsHelp.categories.navigation',
    keyCombo: 'Escape',
    defaultDisplayLabel: 'Esc',
    modes: ['history'],
    customizable: false,
  },
  {
    id: 'settings.exit',
    labelKey: 'shortcutsHelp.items.exitSettings.action',
    categoryKey: 'shortcutsHelp.categories.navigation',
    keyCombo: 'Escape',
    defaultDisplayLabel: 'Esc',
    modes: ['settings'],
    customizable: false,
  },
  {
    id: 'asset.rename.confirm',
    // 附件重命名复用对话框按键，但不重复展示在快捷键帮助列表中。
    labelKey: 'shortcutsHelp.items.confirmRename.action',
    categoryKey: 'shortcutsHelp.categories.dialogs',
    keyCombo: 'Enter',
    defaultDisplayLabel: 'Enter',
    modes: ['dialog'],
    customizable: false,
    showInShortcutsDialog: false,
  },
  {
    id: 'asset.rename.cancel',
    labelKey: 'shortcutsHelp.items.cancelRename.action',
    categoryKey: 'shortcutsHelp.categories.dialogs',
    keyCombo: 'Escape',
    defaultDisplayLabel: 'Esc',
    modes: ['dialog'],
    customizable: false,
    showInShortcutsDialog: false,
  },
]

// 模块加载时建立查找表，保存清单对象引用；之后向清单数组追加条目不会自动进入此表。
const shortcutDefinitionMap = new Map(
  shortcutDefinitions.map((definition) => [definition.id, definition]),
)

// 身份不存在时抛错，成功时返回定义本身，不创建副本或提供默认快捷键。
export function getShortcutDefinition(id: ShortcutId): ShortcutDefinition {
  const definition = shortcutDefinitionMap.get(id)

  if (!definition) {
    throw new Error(`Shortcut definition for "${id}" not found.`)
  }

  return definition
}

// 只保留当前模式且未明确隐藏的条目，模式数少的条目排前，再按翻译键文本排序。
// filter 创建新数组，因此排序不改变集中清单原来的排列。
export function getVisibleShortcutsForMode(
  mode: AppMode,
): ShortcutDefinition[] {
  return shortcutDefinitions
    .filter(
      (definition) =>
        definition.modes.includes(mode) &&
        definition.showInShortcutsDialog !== false,
    )
    .sort((left, right) => {
      if (left.modes.length !== right.modes.length) {
        return left.modes.length - right.modes.length
      }

      return left.labelKey.localeCompare(right.labelKey)
    })
}

// 先去重输入模式，再按定义身份合并条目，避免跨模式快捷键在同一帮助列表重复出现。
// 合并后沿用单模式的排序规则，返回数组独立，内部定义对象仍与清单共享。
export function getVisibleShortcutsForModes(
  modes: AppMode[],
): ShortcutDefinition[] {
  const definitionsById = new Map<ShortcutId, ShortcutDefinition>()

  Array.from(new Set(modes)).forEach((mode) => {
    getVisibleShortcutsForMode(mode).forEach((definition) => {
      definitionsById.set(definition.id, definition)
    })
  })

  return Array.from(definitionsById.values()).sort((left, right) => {
    if (left.modes.length !== right.modes.length) {
      return left.modes.length - right.modes.length
    }

    return left.labelKey.localeCompare(right.labelKey)
  })
}

// Mac 请求且定义提供专用文字时采用专用值，其余情况使用默认显示文字。
export function getShortcutDisplayLabel(id: ShortcutId, isMacOS: boolean) {
  const definition = getShortcutDefinition(id)
  return isMacOS && definition.macDisplayLabel
    ? definition.macDisplayLabel
    : definition.defaultDisplayLabel
}

// 将清单组合、模式与调用方动作装配为处理器描述，enabled 缺省为 true。
// shouldHandle 仅透传给实际事件处理层，此处不执行动作，也不注册或注销处理器。
export function createHotkeyDefinition(
  id: ShortcutId,
  action: () => void,
  options?: Partial<Pick<HotKeyDefinition, 'enabled' | 'shouldHandle'>>,
): HotKeyDefinition {
  const definition = getShortcutDefinition(id)

  return {
    keyCombo: definition.keyCombo,
    enabled: options?.enabled ?? true,
    mode: definition.modes,
    shouldHandle: options?.shouldHandle,
    action,
  }
}
