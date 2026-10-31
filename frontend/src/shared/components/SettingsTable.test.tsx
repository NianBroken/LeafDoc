// 设置表格测试分别验证控件单元格的键盘操作和文字单元格的溢出测量。
// 尺寸由属性与观察器替身指定，测试不依赖 jsdom 实际排版计算宽度。
import { act, cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SettingsTable, SettingsTableCell } from './SettingsTable'
import { TooltipProvider } from './ui/tooltip'
import userEvent from '@testing-library/user-event'

afterEach(() => {
  // 卸载组件并恢复全局替身，确保观察器与 DOM 不跨用例共享。
  cleanup()
  vi.unstubAllGlobals()
})

// 主动调用观察器回调模拟尺寸变化，再检查对应的焦点属性。
describe('SettingsTable', () => {
  it('keeps interactive cells in the native keyboard focus order', async () => {
    // controls 模式直接保留按钮焦点顺序，Enter 触发操作，不套用文字测量容器。
    const onClick = vi.fn()
    const user = userEvent.setup()
    const { container } = render(
      <SettingsTable columns={1}>
        <tbody>
          <tr>
            <SettingsTableCell content="controls">
              <button type="button" onClick={onClick}>
                操作
              </button>
            </SettingsTableCell>
          </tr>
        </tbody>
      </SettingsTable>,
    )
    await user.tab()
    expect(screen.getByRole('button', { name: '操作' })).toHaveFocus()
    await user.keyboard('{Enter}')
    expect(onClick).toHaveBeenCalledOnce()
    expect(container.querySelector('.table-cell-text')).toBeNull()
  })

  it('exposes full overflowing text to keyboard users and releases observers', () => {
    // 捕获 ResizeObserver 回调，分别制造溢出与充足宽度，确认 tabindex 随测量切换。
    let measure = () => {}
    const disconnect = vi.fn()
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(callback: () => void) {
          measure = callback
        }
        observe() {}
        disconnect = disconnect
      },
    )
    const { container, unmount } = render(
      <TooltipProvider>
        <SettingsTable columns={2}>
          <tbody>
            <tr>
              <SettingsTableCell colSpan={2}>完整的长内容</SettingsTableCell>
            </tr>
          </tbody>
        </SettingsTable>
      </TooltipProvider>,
    )
    const text = screen.getByText('完整的长内容')
    Object.defineProperties(text, {
      scrollWidth: { configurable: true, value: 500 },
      clientWidth: { configurable: true, value: 216 },
    })
    act(() => measure())
    expect(text).toHaveAttribute('tabindex', '0')
    expect(container.querySelectorAll('col')).toHaveLength(2)
    expect(container.querySelector('td')).toHaveAttribute('colspan', '2')
    Object.defineProperty(text, 'clientWidth', { value: 600 })
    act(() => measure())
    expect(text).not.toHaveAttribute('tabindex')
    // 卸载必须断开已构造的尺寸观察器，避免组件退出后继续持有被观察元素。
    unmount()
    expect(disconnect).toHaveBeenCalledOnce()
  })
})
