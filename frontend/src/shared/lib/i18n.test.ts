// 语言资源测试检查自称名称、命名空间集合与叶子键集合，不比较不同语言的译文内容。
import { describe, expect, it } from 'vitest'
import { getAvailableLanguages } from '@/shared/lib/i18n'
import de from '@/shared/locales/de/common.json'
import en from '@/shared/locales/en/common.json'

// 与生产语言加载相同地发现两级 JSON 路径，立即取得默认导出的资源对象。
const localeModules = import.meta.glob<{ default: Record<string, unknown> }>(
  '../locales/*/*.json',
  { eager: true },
)

// 按语言和文件名组织资源，后续逐个命名空间比较，避免键名相同但归属文件不同被放过。
const NAMESPACES_BY_LANG: Record<string, Record<string, unknown>> = {}
for (const path in localeModules) {
  const match = /\.\.\/locales\/([^/]+)\/([^/]+)\.json$/.exec(path)
  if (!match) continue
  const [, lang, namespace] = match
  NAMESPACES_BY_LANG[lang] ??= {}
  NAMESPACES_BY_LANG[lang][namespace] = localeModules[path].default
}

// 递归收集非对象值的点分隔路径，空对象没有叶子，数组按其枚举索引遍历。
// 比较对象保存的是路径集合，具体字符串和其他叶子值不参加一致性判断。
function collectKeyPaths(
  value: unknown,
  prefix: string,
  keys: Set<string>,
): void {
  if (typeof value !== 'object' || value === null) {
    keys.add(prefix)
    return
  }
  for (const [key, child] of Object.entries(value)) {
    collectKeyPaths(child, prefix ? `${prefix}.${key}` : key, keys)
  }
}

describe('getAvailableLanguages', () => {
  it('includes English and German with their self-name translated', () => {
    // 核对公开语言列表包含两种语言及各自自称，不要求列表只包含这两项。
    const languages = getAvailableLanguages()

    expect(languages).toContainEqual({ code: 'en', name: 'English' })
    expect(languages).toContainEqual({ code: 'de', name: 'Deutsch' })
  })

  it('reflects language.selfName from each language common.json', () => {
    // 直接核对资源字段，区分资源内容缺失与语言列表构造错误。
    expect(en.language.selfName).toBe('English')
    expect(de.language.selfName).toBe('Deutsch')
  })
})

describe('locale namespace key parity', () => {
  const languages = Object.keys(NAMESPACES_BY_LANG)
  const referenceLang = 'en'

  it('every discovered language ships the same namespace files as English', () => {
    // 排序后比较文件集合，既发现缺失命名空间，也发现额外命名空间。
    const referenceNamespaces = Object.keys(
      NAMESPACES_BY_LANG[referenceLang],
    ).sort()
    for (const lang of languages) {
      if (lang === referenceLang) continue
      expect(Object.keys(NAMESPACES_BY_LANG[lang]).sort()).toEqual(
        referenceNamespaces,
      )
    }
  })

  it.each(Object.keys(NAMESPACES_BY_LANG[referenceLang]))(
    'namespace "%s" has identical keys across all languages',
    (namespace) => {
      // 当前英语命名空间作为参照，每种其他语言必须具有完全相同的叶子路径。
      const referenceKeys = new Set<string>()
      collectKeyPaths(
        NAMESPACES_BY_LANG[referenceLang][namespace],
        '',
        referenceKeys,
      )

      for (const lang of languages) {
        if (lang === referenceLang) continue
        const keys = new Set<string>()
        collectKeyPaths(NAMESPACES_BY_LANG[lang][namespace], '', keys)
        expect(
          [...keys].sort(),
          `locales/${lang}/${namespace}.json is missing or has extra keys compared to English`,
        ).toEqual([...referenceKeys].sort())
      }
    },
  )
})
