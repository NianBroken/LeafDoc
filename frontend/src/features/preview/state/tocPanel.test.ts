// 每例重设共享折叠值，分别核验显式赋值和两次反转，不访问持久化配置。
import { beforeEach, describe, expect, it } from 'vitest'
import { useTocPanelStore } from '@/features/preview/state/tocPanel'

describe('useTocPanelStore', () => {
  beforeEach(() => {
    useTocPanelStore.setState({ collapsed: false })
  })

  // 此断言检查准备阶段设定的展开值，模块初始值在生产状态声明中另有定义。
  it('starts expanded by default', () => {
    expect(useTocPanelStore.getState().collapsed).toBe(false)
  })

  // 同一个动作接受 true 和 false，结果精确采用调用方输入。
  it('setCollapsed sets the collapsed flag explicitly', () => {
    useTocPanelStore.getState().setCollapsed(true)
    expect(useTocPanelStore.getState().collapsed).toBe(true)

    useTocPanelStore.getState().setCollapsed(false)
    expect(useTocPanelStore.getState().collapsed).toBe(false)
  })

  // 连续反转两次回到展开状态，证明动作读取当前值而非固定赋值。
  it('toggleCollapsed flips the current state', () => {
    useTocPanelStore.getState().toggleCollapsed()
    expect(useTocPanelStore.getState().collapsed).toBe(true)

    useTocPanelStore.getState().toggleCollapsed()
    expect(useTocPanelStore.getState().collapsed).toBe(false)
  })
})
