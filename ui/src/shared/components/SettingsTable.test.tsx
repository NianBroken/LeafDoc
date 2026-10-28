import { act, cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SettingsTable, SettingsTableCell } from './SettingsTable'
import { TooltipProvider } from './ui/tooltip'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

// Resize callbacks represent actual overflow; accessible hints follow that state.
describe('SettingsTable', () => {
  it('exposes full overflowing text to keyboard users and releases observers', () => {
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
    unmount()
    expect(disconnect).toHaveBeenCalledOnce()
  })
})
