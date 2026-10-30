// Saving remains automatic while conflict and metadata protections remain active.
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAutoSave } from './useAutoSave'
import { usePageEditorStore } from './pageEditorStore'
import { useDialogsStore } from '@/shared/state/dialogs'
import { ApiLocalizedError } from '@/shared/lib/api/errors'
import type { Page } from '@/shared/lib/api/pages'

vi.mock('sonner', () => ({
  toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() }),
}))
const page = {
  id: 'save-test',
  title: 'Test',
  slug: 'test',
  path: 'test',
  kind: 'page',
  version: 'v1',
  content: 'original',
  tags: [],
  properties: {},
} as Page
const save = vi.fn()

beforeEach(() => {
  vi.useFakeTimers()
  save.mockReset().mockImplementation(async () => {
    usePageEditorStore.setState({ content: page.content })
    return page
  })
  useDialogsStore.setState({ dialogType: null, dialogProps: null })
  usePageEditorStore.setState({
    ...usePageEditorStore.getInitialState(),
    page,
    initialPage: page,
    title: page.title,
    slug: page.slug,
    content: 'changed',
    savePage: save,
  })
})
afterEach(() => {
  vi.useRealTimers()
  usePageEditorStore.setState(usePageEditorStore.getInitialState())
})
describe('always enabled automatic saving', () => {
  it('saves dirty content after the debounce without any preference', async () => {
    const hook = renderHook(() => useAutoSave())
    await act(() => vi.advanceTimersByTimeAsync(2000))
    expect(save).toHaveBeenCalledExactlyOnceWith({ silent: true })
    hook.unmount()
  })
  it('pauses after a version conflict rather than overwriting repeatedly', async () => {
    save.mockRejectedValue(
      new ApiLocalizedError({
        code: 'page_version_conflict',
        message: 'conflict',
        template: 'conflict',
      }),
    )
    const hook = renderHook(() => useAutoSave())
    await act(() => vi.advanceTimersByTimeAsync(2000))
    expect(hook.result.current.status).toBe('paused')
    await act(() => vi.advanceTimersByTimeAsync(10000))
    expect(save).toHaveBeenCalledTimes(1)
    hook.unmount()
  })
  it('does not silently save a changed slug', async () => {
    usePageEditorStore.setState({ slug: 'renamed' })
    const hook = renderHook(() => useAutoSave())
    await act(() => vi.advanceTimersByTimeAsync(4000))
    expect(save).not.toHaveBeenCalled()
    hook.unmount()
  })
})
