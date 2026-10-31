// 本文件收集构建时的全部翻译命名空间，并初始化应用共用的 i18next 实例。
import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
// 加载日期语言同步副作用，日期库监听与翻译实例使用同一个 i18next 对象。
import '@/shared/lib/dateLocale'

// 构建时立即导入两层目录中的 JSON，不在运行中逐个请求词条文件。
// 路径中的语言目录和文件名分别作为语言代码与命名空间。
const localeModules = import.meta.glob<{ default: Record<string, unknown> }>(
  '../locales/*/*.json',
  { eager: true },
)

const LOCALE_PATH_PATTERN = /\.\.\/locales\/([^/]+)\/([^/]+)\.json$/

// 三层映射依次为语言、命名空间和该文件的词条对象。
const resources: Record<string, Record<string, Record<string, unknown>>> = {}

// 仅接收符合路径模式的导入项，未匹配项跳过，不根据文件内容猜测语言。
for (const path in localeModules) {
  const match = LOCALE_PATH_PATTERN.exec(path)
  if (!match) continue
  const [, lang, namespace] = match
  resources[lang] ??= {}
  resources[lang][namespace] = localeModules[path].default
}

// 语言选项包含稳定代码和该语言自身的显示名称。
export type AvailableLanguage = {
  code: string
  name: string
}

// 从 common.language.selfName 读取自称名称，字段缺失时使用代码，空串不会回退。
function selfNameOf(lang: string): string {
  const language = resources[lang]?.common?.language as
    { selfName?: string } | undefined
  return language?.selfName ?? lang
}

// 按语言代码排序并构造独立列表，不按翻译后的显示名称重新排序。
export function getAvailableLanguages(): AvailableLanguage[] {
  return Object.keys(resources)
    .sort()
    .map((code) => ({ code, name: selfNameOf(code) }))
}

// 初始选择中文，缺失翻译回退英语，插值不在 i18next 内执行 HTML 转义。
// React 等实际消费位置仍须按各自上下文处理内容，init 返回值不在本模块等待。
i18next.use(initReactI18next).init({
  lng: 'zh',
  fallbackLng: 'en',
  resources,
  interpolation: {
    escapeValue: false,
  },
})

export default i18next
