// 正文定位测试使用真实 DOM 文本与 mark 节点，关闭布局等待后同步检查首个匹配和清理效果。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  clearSearchQueryHighlights,
  scrollToSearchQuery,
} from '@/shared/lib/scrollToSearchQuery'

// jsdom 未实现 scrollIntoView，测试先在原型上提供该属性，vi.spyOn 才能记录调用。
Element.prototype.scrollIntoView = () => {}

beforeEach(() => {
  // 滚动容器和正文分别创建，正文含两处相同关键词以区分首个匹配与全部匹配。
  const sc = document.createElement('div')
  sc.id = 'scroll-container'
  document.body.appendChild(sc)

  const content = document.createElement('article')
  content.className = 'page-viewer__content'
  content.innerHTML =
    '<p>Hello world</p><p>Find the keyword here please</p><p>keyword again</p>'
  document.body.appendChild(content)
})

afterEach(() => {
  // 每例移除节点并恢复函数替身，不让前一次标记或调用次数进入下一例。
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('scrollToSearchQuery', () => {
  it('highlights and scrolls to the first case-insensitive match', () => {
    // 查询首字母大写，正文小写仍命中；标记必须位于第二段而非第三段。
    const scrollSpy = vi
      .spyOn(HTMLElement.prototype, 'scrollIntoView')
      .mockImplementation(() => {})

    scrollToSearchQuery('Keyword', { waitForStableLayout: false })

    const mark = document.querySelector(
      'mark.search-query-highlight',
    ) as HTMLElement
    expect(mark).not.toBeNull()
    expect(mark.textContent).toBe('keyword')
    expect(scrollSpy).toHaveBeenCalled()

    const content = document.querySelector(
      '.page-viewer__content',
    ) as HTMLElement
    const firstMatchParagraph = content.querySelectorAll('p')[1]
    expect(firstMatchParagraph.contains(mark)).toBe(true)
  })

  it('does nothing for queries shorter than 3 characters', () => {
    // 两字符查询在读取正文前返回，不建立任何高亮标记。
    scrollToSearchQuery('ky', { waitForStableLayout: false })
    expect(document.querySelector('mark.search-query-highlight')).toBeNull()
  })

  it('does nothing when no match exists', () => {
    // 达到长度要求但正文无匹配时，同样不创建标记。
    scrollToSearchQuery('missing', { waitForStableLayout: false })
    expect(document.querySelector('mark.search-query-highlight')).toBeNull()
  })

  it('clears previous highlights before applying a new match', () => {
    // 连续两次定位只保留后一次标记，第一次关键词恢复为普通文本。
    scrollToSearchQuery('keyword', { waitForStableLayout: false })
    expect(
      document.querySelectorAll('mark.search-query-highlight'),
    ).toHaveLength(1)

    scrollToSearchQuery('Hello', { waitForStableLayout: false })
    const marks = document.querySelectorAll('mark.search-query-highlight')
    expect(marks).toHaveLength(1)
    expect(marks[0].textContent).toBe('Hello')
  })

  it('cleanup removes highlights', () => {
    // 调用返回的清理函数后删除当前标记，不要求再次执行搜索。
    const cancel = scrollToSearchQuery('keyword', {
      waitForStableLayout: false,
    })
    expect(document.querySelector('mark.search-query-highlight')).not.toBeNull()
    cancel()
    expect(document.querySelector('mark.search-query-highlight')).toBeNull()
  })
})

describe('clearSearchQueryHighlights', () => {
  it('unwraps highlight marks without losing text', () => {
    // 独立清理入口把 mark 子节点放回原父级，完整句子仍可从正文读取。
    scrollToSearchQuery('keyword', { waitForStableLayout: false })
    clearSearchQueryHighlights()
    expect(document.querySelector('mark.search-query-highlight')).toBeNull()
    expect(document.body.textContent).toContain('Find the keyword here please')
  })
})
