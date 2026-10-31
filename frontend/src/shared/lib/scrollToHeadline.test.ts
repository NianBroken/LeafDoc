// 标题定位测试关闭布局等待，直接检查地址替换分支，滚动调用由元素替身承接。
import { expectDiagnostics } from '@/shared/testutil/expectedDiagnostics'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { scrollToHeadlineHash } from '@/shared/lib/scrollToHeadline'

let replaceStateSpy: ReturnType<typeof vi.spyOn>

beforeEach(() => {
  replaceStateSpy = vi
    .spyOn(history, 'replaceState')
    .mockImplementation(() => {})
  // 实现要求滚动容器存在才继续处理片段，每例都提供独立容器。
  const sc = document.createElement('div')
  sc.id = 'scroll-container'
  document.body.appendChild(sc)
})

afterEach(() => {
  // 清除构造的 DOM 并恢复历史接口，防止后续用例继承标题或调用记录。
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

// 返回带目标 ID 的真实标题元素，只有浏览器滚动动作替换为空实现。
function addHeading(id: string): HTMLElement {
  const el = document.createElement('h2')
  el.id = id
  el.scrollIntoView = vi.fn()
  document.body.appendChild(el)
  return el
}

describe('scrollToHeadlineHash — hash update', () => {
  it('calls history.replaceState with the hash when heading is found', () => {
    // 命中标题时更新片段，状态参数固定为 null，不建立新的历史项。
    addHeading('my-heading')
    scrollToHeadlineHash('#my-heading', { waitForStableLayout: false })
    expect(replaceStateSpy).toHaveBeenCalledWith(null, '', '#my-heading')
  })

  it('does not call replaceState when heading is not found', () => {
    // 缺失目标的告警是本例核对的预期输出，地址替换函数必须保持未调用。
    expectDiagnostics('warn', [['Headline with id "nonexistent" not found.']])
    scrollToHeadlineHash('#nonexistent', { waitForStableLayout: false })
    expect(replaceStateSpy).not.toHaveBeenCalled()
  })

  it('passes encoded hash to replaceState as-is', () => {
    // 查找标题时解码空格，写地址时仍保留传入的编码片段。
    addHeading('hello world')
    scrollToHeadlineHash('#hello%20world', { waitForStableLayout: false })
    expect(replaceStateSpy).toHaveBeenCalledWith(null, '', '#hello%20world')
  })
})
