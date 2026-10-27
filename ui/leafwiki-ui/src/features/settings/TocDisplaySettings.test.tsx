import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, it, expect, vi } from 'vitest'
import { useConfigStore } from '@/stores/config'
import { fetchWithAuth } from '@/lib/api/auth'
import TocDisplaySettings from './TocDisplaySettings'
vi.mock('@/lib/api/auth', () => ({ fetchWithAuth: vi.fn() }))
vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))
beforeEach(() => {
  vi.clearAllMocks()
  useConfigStore.setState({
    tocMode: 'auto',
    loadConfig: vi.fn().mockResolvedValue(undefined),
  })
})
it('offers all three global defaults', () => {
  render(<TocDisplaySettings />)
  expect(screen.getAllByRole('radio')).toHaveLength(3)
  expect(screen.getByRole('radio', { name: /Automatic/ })).toBeChecked()
  expect(
    screen.getByText('Show the table of contents when a page has more than three headings. Hide it on shorter pages.'),
  ).toBeInTheDocument()
})
it('saves the never mode globally', async () => {
  vi.mocked(fetchWithAuth).mockResolvedValue({ mode: 'never' })
  render(<TocDisplaySettings />)
  await userEvent.click(
    screen.getByRole('radio', { name: /Always hide contents/ }),
  )
  await waitFor(() =>
    expect(fetchWithAuth).toHaveBeenCalledWith(
      '/api/admin/settings/toc-display',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ mode: 'never' }),
      }),
    ),
  )
})
