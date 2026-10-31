// 使用可主动通知的媒体查询替身检查主题响应及卸载清理，不依赖宿主系统实际配色。
import { act, renderHook, cleanup } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useDesignModeStore } from './designmode'
import { useResolvedDesignMode } from './useResolvedDesignMode'

// 每个用例结束时卸载钩子、恢复全局对象并重设共享主题状态。
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  useDesignModeStore.setState({ mode: 'light' })
})

describe('useResolvedDesignMode', () => {
  it('follows live system colors and preserves an explicit selection', () => {
    let matches = true
    // 监听集合同时用于发送系统变化通知和检查卸载后是否仍持有回调。
    const listeners = new Set<() => void>()
    vi.stubGlobal('matchMedia', () => ({
      get matches() {
        return matches
      },
      addEventListener: (_event: string, listener: () => void) =>
        listeners.add(listener),
      removeEventListener: (_event: string, listener: () => void) =>
        listeners.delete(listener),
    }))
    useDesignModeStore.setState({ mode: 'system' })
    const { result, unmount } = renderHook(() => useResolvedDesignMode())
    expect(result.current).toBe('light')
    act(() => {
      matches = false
      listeners.forEach((listener) => listener())
    })
    expect(result.current).toBe('dark')
    act(() => useDesignModeStore.setState({ mode: 'light' }))
    // 显式选择覆盖此前的系统深色结果，卸载后监听集合必须为空。
    expect(result.current).toBe('light')
    unmount()
    expect(listeners.size).toBe(0)
  })
})
