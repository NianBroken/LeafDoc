// 通过预置树节点验证标题查询的大小写处理、同名结果和空结果。
import { describe, it, expect, beforeEach } from 'vitest'
import { useTreeStore } from '@/features/tree/state/tree'
import type { PageNode } from '@/shared/lib/api/pages'

const makeNode = (id: string, title: string, path: string): PageNode => ({
  // 测试路径末段作为 slug，节点不包含子项。
  id,
  title,
  slug: path.split('/').pop() ?? path,
  path,
  version: '1',
  kind: 'page',
  children: null,
})

describe('useTreeStore — getPagesByTitle', () => {
  beforeEach(() => {
    // 清空查询涉及的索引与树，避免各用例读取共享状态中的旧节点。
    useTreeStore.setState({
      byId: {},
      byPath: {},
      tree: null,
      flatPages: [],
    })
  })

  const load = (nodes: PageNode[]) => {
    // 直接写入两份索引，测试不经过服务端树加载和父节点补全流程。
    const byId: Record<string, PageNode> = {}
    const byPath: Record<string, PageNode> = {}
    for (const n of nodes) {
      byId[n.id] = n
      byPath[n.path] = n
    }
    useTreeStore.setState({ byId, byPath })
  }

  it('returns a matching page case-insensitively', () => {
    // 同一标题以不同大小写查询，验证匹配按双方小写化后的完整文本进行。
    load([makeNode('1', 'Getting Started', 'docs/getting-started')])
    const { getPagesByTitle } = useTreeStore.getState()
    expect(getPagesByTitle('Getting Started')).toHaveLength(1)
    expect(getPagesByTitle('getting started')).toHaveLength(1)
    expect(getPagesByTitle('GETTING STARTED')).toHaveLength(1)
  })

  it('returns all pages that share a title', () => {
    // 相同标题对应不同路径与 ID，查询应保留两个节点。
    load([
      makeNode('1', 'Notes', 'team/notes'),
      makeNode('2', 'Notes', 'personal/notes'),
    ])
    const { getPagesByTitle } = useTreeStore.getState()
    expect(getPagesByTitle('Notes')).toHaveLength(2)
  })

  it('returns an empty array when no page matches', () => {
    load([makeNode('1', 'Intro', 'intro')])
    const { getPagesByTitle } = useTreeStore.getState()
    expect(getPagesByTitle('Nonexistent')).toHaveLength(0)
  })

  it('returns an empty array when the tree is empty', () => {
    const { getPagesByTitle } = useTreeStore.getState()
    expect(getPagesByTitle('Anything')).toHaveLength(0)
  })
})
