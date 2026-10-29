import { act, renderHook, cleanup } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useDesignModeStore } from './designmode'
import { useResolvedDesignMode } from './useResolvedDesignMode'

// System changes must update every subscriber and unsubscribe on unmount.
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  useDesignModeStore.setState({ mode: 'light' })
})

describe('useResolvedDesignMode', () => {
  it('follows live system colors and preserves an explicit selection', () => {
    let matches = true
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
    expect(result.current).toBe('light')
    unmount()
    expect(listeners.size).toBe(0)
  })
})
