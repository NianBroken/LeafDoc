// 在 Markdown 渲染之前处理双中括号链接，并提供按小写标题查找页面的索引。
// 页面数据由调用方传入，此模块不发请求、不读取路由，也不确认目标页面的访问权限。
import { PageNode } from '@/shared/lib/api/pages'

// 目标首字符不能是空白，后续目标不含竖线、井号、右括号或换行。
// 可选竖线后面的别名至少一个字符，完整匹配后才对目标和别名去掉两端空白。
const WIKILINK_RE = /\[\[(\S[^\]|#\n]*?)(?:\|([^\]\n]+?))?\]\]/g

// 识别三反引号包围的片段和单反引号包围的非空单行片段，不承担完整 Markdown 语法解析。
// 捕获组使代码片段位于结果数组的奇数索引，偶数索引才参与维基链接替换。
const CODE_SPLIT_RE = /(```[\s\S]*?```|`[^`\n]+`)/g

/**
 * 在内容交给 Markdown 渲染器前，将 [[Title]] 语法替换为标准 Markdown 链接。
 *
 * 正则识别的代码片段原样保留，其他片段逐个处理链接；查找结果只在这一轮调用内缓存。
 * 缓存键使用小写标题，因此只有大小写不同的目标也共享第一次查找结果。
 *
 * 唯一命中使用页面 path 生成站内链接，别名只影响显示文字。
 * 多个命中生成 wikilink-ambiguous: 地址，零个命中且不含斜线时生成 wikilink-notfound: 地址。
 * 两种专用地址都对目标编码，具体交互与显示由消费链接的渲染组件处理。
 * 零个命中且目标含斜线时把目标作为路由提示，以尖括号包围地址，保留含空格的路径。
 * 斜线不跳过标题查找，例如 TCP/IP 有唯一标题命中时仍使用页面记录中的 path。
 */
export function preprocessWikilinks(
  content: string,
  getPagesByTitle: (title: string) => PageNode[],
): string {
  // 缓存同时保存空数组和歧义数组，避免相同标题多次调用外部查找函数。
  const cache = new Map<string, PageNode[]>()

  const getMatches = (title: string): PageNode[] => {
    const key = title.toLowerCase()
    if (!cache.has(key)) cache.set(key, getPagesByTitle(title))
    return cache.get(key)!
  }

  const replaceWikilinks = (text: string): string =>
    text.replace(WIKILINK_RE, (_raw, target: string, alias?: string) => {
      const trimmedTarget = target.trim()
      const displayText = alias ? alias.trim() : trimmedTarget

      const matches = getMatches(trimmedTarget)

      // 页面路径直接接在一个前导斜线之后，此分支不编码路径或显示文字。
      if (matches.length === 1) {
        return `[${displayText}](/${matches[0].path})`
      }

      if (matches.length === 0) {
        // 未命中标题时才考虑路径提示，去掉所有前导斜线，再由模板加回唯一斜线。
        // 这样 //host 不会成为省略协议的外部地址，尖括号编码后不能提前结束地址边界。
        // 只有斜线构成的目标会被处理为空路径，因此仍进入未找到标题的专用地址分支。
        const routePath = trimmedTarget
          .replace(/^\/+/, '')
          .replace(/</g, '%3C')
          .replace(/>/g, '%3E')
        if (trimmedTarget.includes('/') && routePath !== '') {
          return `[${displayText}](</${routePath}>)`
        }
        return `[${displayText}](wikilink-notfound:${encodeURIComponent(trimmedTarget)})`
      }

      // 保留原始目标文字供后续歧义选择使用，不在预处理阶段替用户选择其中一页。
      return `[${displayText}](wikilink-ambiguous:${encodeURIComponent(trimmedTarget)})`
    })

  // 捕获的代码片段位于奇数索引，直接拼回原文，只有偶数索引参加链接替换。
  return content
    .split(CODE_SPLIT_RE)
    .map((segment, i) => (i % 2 === 0 ? replaceWikilinks(segment) : segment))
    .join('')
}

/**
 * 按 Object.values 的枚举顺序收集所有节点，同名标题保留多个节点和各自对象引用。
 * 此函数不遍历 children，也不排序；节点顺序由传入的 byId 对象决定。
 * 小写转换不去掉两端空白，Map 允许 __proto__ 等文本作为普通索引键。
 */
export function buildWikiTitleIndex(
  byId: Record<string, PageNode>,
): Map<string, PageNode[]> {
  const index = new Map<string, PageNode[]>()
  for (const node of Object.values(byId)) {
    const title = node.title.toLowerCase()
    const matches = index.get(title)
    // 已有标题继续追加，同一标题的候选顺序与输入枚举顺序一致。
    if (matches) matches.push(node)
    else index.set(title, [node])
  }
  return index
}
