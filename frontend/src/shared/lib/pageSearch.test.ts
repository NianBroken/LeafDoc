// 使用同一组标题与路径交叉出现的页面核对搜索评分，不读取页面正文或调用服务端搜索。
import { describe, expect, it } from 'vitest'
import {
  buildFlatPageSearchItems,
  searchFlatPageSearchItems,
} from '@/shared/lib/pageSearch'
import type { PageNode } from '@/shared/lib/api/pages'
// 单节点没有子项，根节点只承载候选列表，空路径的根本身不进入搜索项。
const node = (id: string, title: string, path: string): PageNode => ({
  id,
  title,
  path,
  slug: id,
  version: 'v1',
  kind: 'page',
  children: null,
})
const root: PageNode = {
  ...node('root', 'Root', ''),
  children: [
    node('1', 'Alpha Beta', 'alpha-beta'),
    node('2', 'Alpha', 'beta/alpha'),
    node('3', 'Gamma', 'alpha/beta'),
    node('4', 'Other', 'elsewhere'),
  ],
}
describe('searchFlatPageSearchItems', () => {
  it('preserves multiword fallback ranking and case/whitespace normalization', () => {
    // 查询内部多空格不折叠为单空格，因此三项依靠分词全命中并按标题排序。
    const items = buildFlatPageSearchItems(root)
    expect(
      searchFlatPageSearchItems(items, '  ALPHA   beta  ').map((n) => n.id),
    ).toEqual(['2', '1', '3'])
    // 单空格查询直接命中完整标题，有限结果只保留分数最高的一项。
    expect(
      searchFlatPageSearchItems(items, 'alpha beta', 1).map((n) => n.id),
    ).toEqual(['1'])
  })
  it('preserves title ranking, empty queries, and zero limits', () => {
    // 精确标题优先于标题前缀和路径前缀，空查询按输入顺序截取。
    const items = buildFlatPageSearchItems(root)
    expect(searchFlatPageSearchItems(items, 'alpha').map((n) => n.id)).toEqual([
      '2',
      '1',
      '3',
    ])
    expect(searchFlatPageSearchItems(items, '', 2).map((n) => n.id)).toEqual([
      '1',
      '2',
    ])
    // 零上限和没有匹配词分别返回空数组，不以任意候选填满列表。
    expect(searchFlatPageSearchItems(items, 'alpha', 0)).toEqual([])
    expect(searchFlatPageSearchItems(items, 'missing words')).toEqual([])
  })
})
