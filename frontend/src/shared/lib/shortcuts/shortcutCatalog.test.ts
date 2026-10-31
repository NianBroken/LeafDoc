// 快捷键清单测试核对身份、模式筛选、展示排序与处理器装配，不派发真实键盘事件。
import { describe, expect, it } from 'vitest'
import {
  createHotkeyDefinition,
  getShortcutDefinition,
  getShortcutDisplayLabel,
  getVisibleShortcutsForMode,
  getVisibleShortcutsForModes,
  shortcutDefinitions,
} from '@/shared/lib/shortcuts/shortcutCatalog'

describe('shortcutCatalog', () => {
  it('defines unique shortcut ids', () => {
    // 集合大小与原数组相同才表示没有重复身份，避免查找表悄然覆盖定义。
    const ids = shortcutDefinitions.map((definition) => definition.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('returns the registered definition for a known shortcut', () => {
    // 同时核对组合、模式和翻译键，确认身份查找返回同一个业务条目。
    expect(getShortcutDefinition('page.quickSwitcher.open')).toMatchObject({
      id: 'page.quickSwitcher.open',
      keyCombo: 'Mod+Alt+KeyP',
      modes: ['view'],
      labelKey: 'shortcutsHelp.items.goToPage.action',
    })
  })

  it('filters visible shortcuts by the current mode', () => {
    // 完整列表同时约束可见条目和排序，分别覆盖查看、编辑与历史模式。
    expect(getVisibleShortcutsForMode('view').map((item) => item.id)).toEqual([
      'viewer.page.copy',
      'viewer.page.delete',
      'viewer.page.edit',
      'page.quickSwitcher.open',
      'viewer.page.history',
      'viewer.page.print',
      'viewer.page.permalink',
      'viewer.toc.toggle',
      'sidebar.explorer.open',
      'sidebar.search.open',
      'shortcuts.help.open',
    ])

    expect(getVisibleShortcutsForMode('edit').map((item) => item.id)).toEqual([
      'editor.page.close',
      'editor.format.bold',
      'editor.format.inlineCode',
      'editor.format.italic',
      'editor.heading.one',
      'editor.heading.three',
      'editor.heading.two',
      'editor.link.insert',
      'editor.page.save',
      'sidebar.explorer.open',
      'sidebar.search.open',
      'shortcuts.help.open',
    ])

    expect(
      getVisibleShortcutsForMode('history').map((item) => item.id),
    ).toEqual([
      'history.page.close',
      'sidebar.explorer.open',
      'sidebar.search.open',
      'shortcuts.help.open',
    ])
  })

  it('does not advertise the Explorer/Search panel toggles on the Settings page (#1550)', () => {
    // 设置模式的清单只列退出与帮助，侧栏切换定义的 modes 不包含 settings。
    expect(
      getVisibleShortcutsForMode('settings').map((item) => item.id),
    ).toEqual(['settings.exit', 'shortcuts.help.open'])
  })

  it('includes the remaining viewer and editor shortcuts in the catalog', () => {
    // 直接查找常用业务身份，核对帮助、查看动作与编辑格式命令的默认组合。
    expect(getShortcutDefinition('shortcuts.help.open').keyCombo).toBe(
      'Mod+Slash',
    )
    expect(getShortcutDefinition('viewer.page.edit').keyCombo).toBe('Mod+KeyE')
    expect(getShortcutDefinition('viewer.page.history').keyCombo).toBe(
      'Mod+KeyH',
    )
    expect(getShortcutDefinition('editor.format.bold').keyCombo).toBe(
      'Mod+KeyB',
    )
    expect(getShortcutDefinition('editor.link.insert').keyCombo).toBe(
      'Mod+KeyK',
    )
  })

  it('returns platform-specific display labels', () => {
    // 相同身份按平台标记显示 Ctrl 或 Cmd，未提供 Mac 专用值时仍使用默认值。
    expect(getShortcutDisplayLabel('page.quickSwitcher.open', false)).toBe(
      'Ctrl+Alt+P',
    )
    expect(getShortcutDisplayLabel('page.quickSwitcher.open', true)).toBe(
      'Cmd+Option+P',
    )
    expect(getShortcutDisplayLabel('sidebar.explorer.open', false)).toBe(
      'Ctrl+Shift+E',
    )
    expect(getShortcutDisplayLabel('shortcuts.help.open', false)).toBe('Ctrl+/')
    expect(getShortcutDisplayLabel('shortcuts.help.open', true)).toBe('Cmd+/')
    expect(getShortcutDisplayLabel('viewer.page.delete', false)).toBe(
      'Ctrl+Delete',
    )
    expect(getShortcutDisplayLabel('editor.heading.one', true)).toBe(
      'Cmd+Option+1',
    )
  })

  it('creates a hotkey definition with the catalog combo and modes', () => {
    // 未提供选项时启用处理器，组合和模式从清单复制到注册描述。
    const hotkey = createHotkeyDefinition('history.page.close', () => undefined)

    expect(hotkey).toMatchObject({
      keyCombo: 'Escape',
      enabled: true,
      mode: ['history'],
    })
  })

  it('supports generic dialog and asset rename shortcuts', () => {
    // 通用对话框与附件重命名各有身份，但分别共用 Enter 和 Escape 按键。
    expect(getShortcutDefinition('dialog.close').keyCombo).toBe('Escape')
    expect(getShortcutDefinition('dialog.confirm').keyCombo).toBe('Enter')
    expect(getShortcutDefinition('asset.rename.confirm').keyCombo).toBe('Enter')
    expect(getShortcutDefinition('asset.rename.cancel').keyCombo).toBe('Escape')
  })

  it('can merge page-mode and dialog shortcuts without duplicates', () => {
    // 历史模式与对话框模式合并后保留各身份一次，隐藏的重命名条目不出现。
    expect(
      getVisibleShortcutsForModes(['history', 'dialog']).map((item) => item.id),
    ).toEqual([
      'history.page.close',
      'dialog.close',
      'dialog.confirm',
      'sidebar.explorer.open',
      'sidebar.search.open',
      'shortcuts.help.open',
    ])
  })
})
