// 本文件验证最近 Wiki 位置的内存存储，不建立浏览器路由，也不检查页面跳转结果。
// 路径、查询和导航状态一起保存，清除时整体返回 null。
import { beforeEach, describe, expect, it } from 'vitest'
import { useLastWikiLocationStore } from '@/shared/state/lastWikiLocation'

// 每例都从空位置开始，避免前一个用例保存的状态影响初始值断言。
beforeEach(() => {
  useLastWikiLocationStore.setState({ location: null })
})

describe('useLastWikiLocationStore', () => {
  // 初始化后的 location 是 null，不用空路径对象替代未记录状态。
  it('starts with no remembered location', () => {
    expect(useLastWikiLocationStore.getState().location).toBeNull()
  })

  // 导航访问标识放在 state 内，断言整个对象，确认保存路径时没有丢失该标识。
  it('stores and returns a location including its state', () => {
    const loc = {
      pathname: '/docs/getting-started',
      search: '?x=1',
      state: { leafdocVisitId: 'abc-123' },
    }

    useLastWikiLocationStore.getState().setLocation(loc)

    expect(useLastWikiLocationStore.getState().location).toEqual(loc)
  })

  // 先保存真实位置再清除，使测试确实经过非空到空的状态变化。
  it('can be reset back to null', () => {
    useLastWikiLocationStore.getState().setLocation({
      pathname: '/a',
      search: '',
      state: null,
    })

    useLastWikiLocationStore.getState().setLocation(null)

    expect(useLastWikiLocationStore.getState().location).toBeNull()
  })
})
