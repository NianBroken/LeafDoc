// 本文件把界面语言映射为 date-fns 语言对象和 Intl 语言标签。
// 模块加载时立即设置日期库默认语言并注册语言变化监听，之后由事件更新。
// i18n.ts 的静态导入在它自身初始化代码执行前加载本模块，不是在 init 之后才注册。

import { setDefaultOptions } from 'date-fns'
import { de, enUS, es, fr, zhCN, type Locale } from 'date-fns/locale'
import i18next from 'i18next'

// 界面短代码与日期库对象的对应关系，未知代码回退为美式英语。
const DATE_FNS_LOCALES: Record<string, Locale> = {
  en: enUS,
  de,
  es,
  fr,
  zh: zhCN,
}

// Intl 使用的区域标签，已列出的短代码会采用这里明确指定的区域。
const BCP47_TAGS: Record<string, string> = {
  en: 'en-US',
  de: 'de-DE',
  zh: 'zh-CN',
  es: 'es-ES',
  fr: 'fr-FR',
}

// 优先使用显式参数，其次当前语言和中文，按连字符取首段，不转换大小写。
function shortCode(lng?: string): string {
  return (lng ?? i18next.language ?? 'zh').split('-')[0]
}

/** 返回指定或当前语言的日期库对象，未匹配时回退 enUS。 */
export function dateFnsLocale(lng?: string): Locale {
  return DATE_FNS_LOCALES[shortCode(lng)] ?? enUS
}

/** 已知短代码返回配置区域标签，未知代码原样返回选中的完整语言文本。 */
export function bcp47(lng?: string): string {
  const raw = lng ?? i18next.language ?? 'zh'
  return BCP47_TAGS[shortCode(raw)] ?? raw
}

// 修改 date-fns 的全局默认选项，不改变 i18next 本身的语言。
function applyDateFnsLocale(lng?: string): void {
  setDefaultOptions({ locale: dateFnsLocale(lng) })
}

// 此监听服务整个模块生命周期，没有组件级卸载步骤。
applyDateFnsLocale()
i18next.on('languageChanged', applyDateFnsLocale)
