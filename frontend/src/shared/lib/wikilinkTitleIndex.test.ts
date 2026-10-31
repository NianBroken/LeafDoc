// 标题索引测试核对小写键、同名候选顺序和索引对象之间的独立性。
import { describe, it, expect } from 'vitest'
import {
  buildWikiTitleIndex,
  preprocessWikilinks,
} from '@/shared/lib/preprocessWikilinks'
import type { PageNode } from '@/shared/lib/api/pages'
// 提供构造索引所需的完整节点形状，路径与标识相同以便直接检查预处理后的链接。
const page = (id: string, title: string): PageNode => ({
  id,
  title,
  path: id,
  slug: id,
  version: 'v1',
  kind: 'page',
  children: null,
})
describe('buildWikiTitleIndex', () => {
  it('preserves Unicode case folding, duplicate order and special map keys', () => {
    // 两个德语标题的小写结果相同，__proto__ 在 Map 中仍作为普通标题保存。
    const first = page('1', 'Überblick'),
      second = page('2', 'ÜBERBLICK'),
      special = page('3', '__proto__')
    const index = buildWikiTitleIndex({ first, second, special })
    expect(index.get('überblick')).toEqual([first, second])
    expect(index.get('__proto__')).toEqual([special])
    expect(index.get('missing')).toBeUndefined()
    // 同名目标保持歧义地址，唯一命中的特殊标题则使用对应节点路径。
    expect(
      preprocessWikilinks(
        '[[Überblick]] [[__proto__]]',
        (title) => index.get(title.toLowerCase()) ?? [],
      ),
    ).toBe('[Überblick](wikilink-ambiguous:%C3%9Cberblick) [__proto__](/3)')
  })
  it('rebuilds after a title changes without mutating the old index', () => {
    // 使用浅拷贝节点构造第二个索引，确认它按新标题登记，而第一个索引仍保留旧引用。
    const old = page('1', 'Before')
    const first = buildWikiTitleIndex({ old })
    const next = buildWikiTitleIndex({ old: { ...old, title: 'After' } })
    expect(first.get('before')).toEqual([old])
    expect(next.has('before')).toBe(false)
    expect(next.get('after')?.[0].id).toBe('1')
  })
})
