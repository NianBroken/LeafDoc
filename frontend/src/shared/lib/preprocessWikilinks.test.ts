// 维基链接预处理测试直接核对输出 Markdown，覆盖标题查找、路径提示、代码保护和调用内缓存。
// 查找函数使用固定返回值，测试不访问服务器，也不实际渲染链接或打开歧义对话框。
import { describe, it, expect, vi } from 'vitest'
import { preprocessWikilinks } from '@/shared/lib/preprocessWikilinks'
import type { PageNode } from '@/shared/lib/api/pages'

// 节点路径与显示标题分别给定，用来确认解析结果使用记录路径而非标题文本。
const page = (id: string, path: string, title: string): PageNode => ({
  id,
  title,
  slug: path.split('/').pop() ?? path,
  path,
  version: '1',
  kind: 'page',
  children: null,
})

// 三种替身分别表示没有候选、唯一候选和多个候选，不在替身内部实现标题匹配。
const noMatch = () => []
const singleMatch = (p: PageNode) => () => [p]
const multiMatch = (pages: PageNode[]) => () => pages

describe('preprocessWikilinks', () => {
  describe('single match', () => {
    it('replaces [[Title]] with a resolved markdown link', () => {
      // 唯一候选提供 docs/intro 路径，链接外的普通正文保持原样。
      const docs = page('1', 'docs/intro', 'Intro')
      const result = preprocessWikilinks(
        'See [[Intro]] here.',
        singleMatch(docs),
      )
      expect(result).toBe('See [Intro](/docs/intro) here.')
    })

    it('uses the alias as display text for [[Title|Alias]]', () => {
      // 别名只替换显示文字，目标仍使用同一页面记录中的路径。
      const docs = page('1', 'docs/intro', 'Intro')
      const result = preprocessWikilinks(
        '[[Intro|our intro]]',
        singleMatch(docs),
      )
      expect(result).toBe('[our intro](/docs/intro)')
    })

    it('trims trailing whitespace from target and alias', () => {
      // 目标与别名末尾的空白不进入生成链接，两者分别执行 trim。
      const docs = page('1', 'docs/intro', 'Intro')
      const result = preprocessWikilinks(
        '[[Intro |our intro ]]',
        singleMatch(docs),
      )
      expect(result).toBe('[our intro](/docs/intro)')
    })
  })

  describe('no match', () => {
    it('emits wikilink-notfound: scheme when no page has that title', () => {
      // 没有标题命中且目标没有斜线，结果使用未找到目标的专用地址。
      const result = preprocessWikilinks('[[Missing Page]]', noMatch)
      expect(result).toBe('[Missing Page](wikilink-notfound:Missing%20Page)')
    })

    it('percent-encodes spaces and special characters in the notfound scheme', () => {
      // 显示文字保留空格，地址中的目标文本进行百分号编码。
      const result = preprocessWikilinks('[[Project Plan]]', noMatch)
      expect(result).toBe('[Project Plan](wikilink-notfound:Project%20Plan)')
    })
  })

  describe('multiple matches', () => {
    it('emits wikilink-ambiguous: scheme when multiple pages match', () => {
      // 多个同名候选不在预处理阶段选取，生成携带标题的歧义地址。
      const matches = multiMatch([
        page('1', 'a/notes', 'Notes'),
        page('2', 'b/notes', 'Notes'),
      ])
      const result = preprocessWikilinks('[[Notes]]', matches)
      expect(result).toBe('[Notes](wikilink-ambiguous:Notes)')
    })

    it('encodes the title in the ambiguous scheme', () => {
      // 歧义目标的空格也编码，显示文字仍保持原来的标题。
      const matches = multiMatch([
        page('1', 'a/my-notes', 'My Notes'),
        page('2', 'b/my-notes', 'My Notes'),
      ])
      const result = preprocessWikilinks('[[My Notes]]', matches)
      expect(result).toBe('[My Notes](wikilink-ambiguous:My%20Notes)')
    })
  })

  describe('path hints ([[Folder/Title]])', () => {
    it('converts a slash-containing target with no title match to a direct path link', () => {
      // 没有标题命中时把含斜线的目标用作站内路径提示，地址带尖括号边界。
      const result = preprocessWikilinks('[[docs/intro]]', noMatch)
      expect(result).toBe('[docs/intro](</docs/intro>)')
    })

    it('uses the alias as display text for path hints', () => {
      // 路径提示与标题链接采用同一别名规则，不把别名写入目标路径。
      const result = preprocessWikilinks('[[docs/intro|Introduction]]', noMatch)
      expect(result).toBe('[Introduction](</docs/intro>)')
    })

    // 标题本身包含斜线仍先使用标题查找，唯一命中后不进入路径提示分支。
    it('resolves a slash-containing target by title when a single page matches', () => {
      const guide = page('1', 'dev/c-cpp-guide', 'C/C++ Guide')
      const result = preprocessWikilinks('[[C/C++ Guide]]', singleMatch(guide))
      expect(result).toBe('[C/C++ Guide](/dev/c-cpp-guide)')
    })

    // 含斜线和空格的长标题同样使用页面记录路径，显示标题与目标路径明确分离。
    it('resolves a slash-and-space title against the tree instead of emitting raw text', () => {
      const adr = page(
        '1',
        'guides/optional-smtp',
        'ADR-0011: SMTP as a CLI/ENV-Only Optional Feature',
      )
      const result = preprocessWikilinks(
        '[[ADR-0011: SMTP as a CLI/ENV-Only Optional Feature]]',
        singleMatch(adr),
      )
      expect(result).toBe(
        '[ADR-0011: SMTP as a CLI/ENV-Only Optional Feature](/guides/optional-smtp)',
      )
    })

    // 未命中的含空格路径用尖括号包围，空格保留在地址边界内。
    it('angle-brackets an unresolved slash target so spaces stay parseable', () => {
      const result = preprocessWikilinks('[[some/nested page]]', noMatch)
      expect(result).toBe('[some/nested page](</some/nested page>)')
    })

    // 多个前导斜线归为一个，地址仍指向站内路径而非省略协议的外部主机。
    it('collapses leading slashes in the path-hint destination to exactly one', () => {
      const result = preprocessWikilinks('[[//evil.example/path]]', noMatch)
      expect(result).toBe('[//evil.example/path](</evil.example/path>)')
    })

    it('strips a single leading slash from the path-hint destination', () => {
      // 输入已有单个前导斜线时也只保留一个，显示文字不受路径规范化影响。
      const result = preprocessWikilinks('[[/docs/intro]]', noMatch)
      expect(result).toBe('[/docs/intro](</docs/intro>)')
    })

    // 目标内的尖括号编码，不能结束包围路径的外层尖括号。
    it('percent-escapes angle brackets in the path-hint destination', () => {
      const result = preprocessWikilinks('[[docs/<script>]]', noMatch)
      expect(result).toBe('[docs/<script>](</docs/%3Cscript%3E>)')
    })

    it('treats an ambiguous slash-containing title as ambiguous, not a path hint', () => {
      // 两个 TCP/IP 标题构成歧义，即使带斜线也不能降为直接路径。
      const matches = multiMatch([
        page('1', 'a/tcp-ip', 'TCP/IP'),
        page('2', 'b/tcp-ip', 'TCP/IP'),
      ])
      const result = preprocessWikilinks('[[TCP/IP]]', matches)
      expect(result).toBe('[TCP/IP](wikilink-ambiguous:TCP%2FIP)')
    })
  })

  describe('code block protection', () => {
    it('does not convert [[Title]] inside fenced code blocks', () => {
      // 三反引号片段整体保留，其中的双中括号不参与替换。
      const content = '```\nSee [[Intro]] for details\n```'
      const result = preprocessWikilinks(
        content,
        singleMatch(page('1', 'intro', 'Intro')),
      )
      expect(result).toBe(content)
    })

    it('does not convert [[Title]] inside inline code', () => {
      // 单反引号代码保留原字符，标题查找结果不影响其内容。
      const content = 'Use `[[Title]]` syntax.'
      const result = preprocessWikilinks(
        content,
        singleMatch(page('1', 'title', 'Title')),
      )
      expect(result).toBe(content)
    })

    it('converts [[Title]] outside code blocks but leaves code intact', () => {
      // 同一标题在普通正文中替换，在代码片段内保持原文。
      const intro = page('1', 'docs/intro', 'Intro')
      const content = 'See [[Intro]].\n\n```\n[[Intro]] example\n```'
      const result = preprocessWikilinks(content, singleMatch(intro))
      expect(result).toBe(
        'See [Intro](/docs/intro).\n\n```\n[[Intro]] example\n```',
      )
    })

    it('handles multiple code spans and wiki-links interleaved', () => {
      // 两段行内代码夹住普通链接，确认分割与拼接没有丢失片段或空格。
      const intro = page('1', 'docs/intro', 'Intro')
      const content = '`code1` [[Intro]] `code2`'
      const result = preprocessWikilinks(content, singleMatch(intro))
      expect(result).toBe('`code1` [Intro](/docs/intro) `code2`')
    })
  })

  describe('caching', () => {
    it('calls getPagesByTitle only once for repeated occurrences of the same title', () => {
      // 重复标题共用这一轮缓存，断言外部查找只执行一次。
      const intro = page('1', 'docs/intro', 'Intro')
      const getPagesByTitle = vi.fn().mockReturnValue([intro])
      preprocessWikilinks('[[Intro]] and [[Intro]] again', getPagesByTitle)
      expect(getPagesByTitle).toHaveBeenCalledTimes(1)
    })

    it('calls getPagesByTitle separately for case-variant titles', () => {
      const getPagesByTitle = vi.fn().mockReturnValue([])
      preprocessWikilinks('[[Notes]] and [[notes]]', getPagesByTitle)
      // 缓存键转小写，大小写不同的两个目标也共用一次查找，空结果同样缓存。
      expect(getPagesByTitle).toHaveBeenCalledTimes(1)
    })
  })

  // Bash 条件样本在双中括号后带空白，不满足目标首字符必须非空白的匹配规则。
  describe('bash double-bracket conditionals', () => {
    it('does not treat [[ ]] in bash conditionals as a wikilink', () => {
      // 带变量与逻辑运算的样本保持整段原文，不生成未找到标题的地址。
      const content = 'if [[ -n "$computed_hash" && -n "$src_hash" ]]; then'
      const result = preprocessWikilinks(content, noMatch)
      expect(result).toBe(content)
    })

    it('does not treat [[ ]] with == operator as a wikilink', () => {
      // 等号比较样本同样由开头空白排除，不需要识别 Bash 运算符。
      const content = 'if [[ "$src_hash" == sha256-* ]]; then'
      const result = preprocessWikilinks(content, noMatch)
      expect(result).toBe(content)
    })

    // 普通标题只要在左括号后有空白也不匹配，区分条件是文本结构而非内容语言。
    it('does not treat [[ Title ]] with leading space as a wikilink', () => {
      const content = '[[ Intro ]]'
      const result = preprocessWikilinks(content, noMatch)
      expect(result).toBe(content)
    })
  })

  describe('edge cases', () => {
    it('returns empty string unchanged', () => {
      // 空正文没有可替换片段，拼接结果仍为空字符串。
      expect(preprocessWikilinks('', noMatch)).toBe('')
    })

    it('returns content without wiki-links unchanged', () => {
      // 标准 Markdown 链接不属于双中括号语法，全文保持原值。
      const content = 'Normal [link](/page) and text.'
      expect(preprocessWikilinks(content, noMatch)).toBe(content)
    })

    it('handles multiple different wiki-links in one pass', () => {
      // 不同标题各自取得不同记录，两个替换结果按正文出现顺序拼接。
      const pageA = page('1', 'a', 'Alpha')
      const pageB = page('2', 'b', 'Beta')
      const lookup = (title: string) => {
        if (title === 'Alpha') return [pageA]
        if (title === 'Beta') return [pageB]
        return []
      }
      const result = preprocessWikilinks('[[Alpha]] and [[Beta]]', lookup)
      expect(result).toBe('[Alpha](/a) and [Beta](/b)')
    })
  })
})
