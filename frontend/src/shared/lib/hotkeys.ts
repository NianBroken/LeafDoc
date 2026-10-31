// 本文件将快捷键声明和键盘事件归一为同一种文本，供注册与匹配复用。
// 修饰键按固定顺序输出，Ctrl 与 Meta 均归并为 Mod。
const modifierOrder = ['Mod', 'Shift', 'Alt'] as const

// 输入别名按小写查询，平台常见命名映射到三个统一修饰键。
const modifierAliases: Record<string, (typeof modifierOrder)[number]> = {
  alt: 'Alt',
  command: 'Mod',
  cmd: 'Mod',
  control: 'Mod',
  ctrl: 'Mod',
  meta: 'Mod',
  mod: 'Mod',
  option: 'Alt',
  shift: 'Shift',
}

// 少量命名键统一拼写，未列出的名称不会自动推断为其他键。
const keyAliases: Record<string, string> = {
  del: 'Delete',
  delete: 'Delete',
  enter: 'Enter',
  esc: 'Escape',
  escape: 'Escape',
  space: 'Space',
  spacebar: 'Space',
  tab: 'Tab',
}

// 只识别空格、反引号、斜线、字母物理键和数字物理键，其余 code 返回空串。
function normalizeEventCode(code: string) {
  if (code === 'Space') {
    return 'Space'
  }

  if (code === 'Backquote') {
    return 'Backquote'
  }

  if (code === 'Slash') {
    return 'Slash'
  }

  if (/^Key[A-Za-z]$/.test(code)) {
    return `Key${code.slice(3).toUpperCase()}`
  }

  if (/^Digit\d$/.test(code)) {
    return code
  }

  return ''
}

// 将单键文本按别名、物理 code、单字母、单数字顺序处理，其余去掉两端空白后保留。
// 不在这里检查输入法组合状态，也不要求返回值一定属于浏览器支持的按键名。
function normalizeHotkeyKey(key: string) {
  const trimmedKey = key.trim()
  if (!trimmedKey) {
    return ''
  }

  const aliasedKey = keyAliases[trimmedKey.toLowerCase()]
  if (aliasedKey) {
    return aliasedKey
  }

  const normalizedCode = normalizeEventCode(trimmedKey)
  if (normalizedCode) {
    return normalizedCode
  }

  if (/^[A-Za-z]$/.test(trimmedKey)) {
    return `Key${trimmedKey.toUpperCase()}`
  }

  if (/^\d$/.test(trimmedKey)) {
    return `Digit${trimmedKey}`
  }

  return trimmedKey
}

// 用加号拆分组合，丢弃空段，去重修饰键并按固定顺序输出。
// 普通键保持出现顺序且不去重，文本形式的加号键自身无法通过此拆分规则表达。
export function normalizeHotkeyCombo(keyCombo: string) {
  const parts = keyCombo
    .split('+')
    .map((part) => part.trim())
    .filter(Boolean)

  const modifierSet = new Set<(typeof modifierOrder)[number]>()
  const keys: string[] = []

  for (const part of parts) {
    const modifier = modifierAliases[part.toLowerCase()]
    if (modifier) {
      modifierSet.add(modifier)
      continue
    }

    keys.push(normalizeHotkeyKey(part))
  }

  return [
    ...modifierOrder.filter((modifier) => modifierSet.has(modifier)),
    ...keys,
  ]
    .filter(Boolean)
    .join('+')
}

// 从元素自身或最近祖先的 data-allow-hotkeys 读取空白分隔的允许列表。
// 仅规范列表中的声明，comboString 应由调用方预先规范；缺少属性或空值返回 false。
export function isHotkeyAllowedOnElement(
  element: Element | null,
  comboString: string,
) {
  const allowedCombos = element
    ?.closest('[data-allow-hotkeys]')
    ?.getAttribute('data-allow-hotkeys')

  if (!allowedCombos) {
    return false
  }

  return allowedCombos
    .split(/\s+/)
    .filter(Boolean)
    .some((combo) => normalizeHotkeyCombo(combo) === comboString)
}

// 先根据事件布尔字段收集修饰键，再优先用支持的物理 code，其他按键使用 key 文本。
// 此函数只生成组合文本，不阻止事件默认行为，不检查 repeat 或 isComposing。
export function getHotkeyComboFromEvent(event: KeyboardEvent) {
  const parts: string[] = []

  if (event.ctrlKey || event.metaKey) parts.push('Mod')
  if (event.shiftKey) parts.push('Shift')
  if (event.altKey) parts.push('Alt')

  const normalizedCode = normalizeEventCode(event.code)
  if (normalizedCode) {
    parts.push(normalizedCode)
  } else {
    parts.push(normalizeHotkeyKey(event.key))
  }

  return normalizeHotkeyCombo(parts.join('+'))
}
