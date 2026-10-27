import { ApiLocalizedError } from '@/shared/lib/api/errors'
import type { PageNode } from '@/shared/lib/api/pages'
import { useResyncStore } from '@/features/settings/state/resync'
import { useSessionStore } from '@/shared/state/session'
import { useTreeStore } from '@/features/tree/state/tree'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type React from 'react'
import { MemoryRouter } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import TreeView from '@/features/tree/TreeView'

vi.mock('react-i18next', () => ({
  initReactI18next: { type: '3rdParty', init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock('@/shared/components/TooltipWrapper', () => ({
  TooltipWrapper: ({ children }: { children: React.ReactNode }) => children,
}))

vi.mock('@/shared/components/ui/accordion', () => ({
  Accordion: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}))

vi.mock('@/shared/components/ui/dropdown-menu', () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: React.ReactNode
    onClick?: () => void
  }) => <button onClick={onClick}>{children}</button>,
  DropdownMenuTrigger: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}))

vi.mock('@/features/sidebar/SidebarAccordionSection', () => ({
  SidebarAccordionSection: ({
    children,
    actions,
  }: {
    children: React.ReactNode
    actions?: React.ReactNode
  }) => (
    <div>
      <div>{actions}</div>
      <div>{children}</div>
    </div>
  ),
}))

vi.mock('@/features/favorites/FavoritesSection', () => ({
  FavoritesSection: () => <div />,
}))

vi.mock('@/features/tree/PinnedSection', () => ({
  PinnedSection: () => <div />,
}))

vi.mock('@/features/tree/TreeDnd', () => ({
  TreeDndProvider: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}))

vi.mock('@/features/tree/TreeNode', () => ({
  TreeNode: () => <div />,
}))

const { triggerResyncMock, getResyncStatusMock, toastMock } = vi.hoisted(
  () => ({
    triggerResyncMock: vi.fn(),
    getResyncStatusMock: vi.fn(),
    toastMock: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
  }),
)

vi.mock('@/shared/lib/api/resync', () => ({
  triggerResync: () => triggerResyncMock(),
  getResyncStatus: () => getResyncStatusMock(),
}))

vi.mock('@/shared/lib/sleep', () => ({
  sleep: () => Promise.resolve(),
}))

vi.mock('sonner', () => ({ toast: toastMock }))

const rootNode: PageNode = {
  id: 'root',
  title: 'root',
  slug: '',
  path: '',
  version: '1',
  children: [],
  kind: 'section',
}

function renderTreeView() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <TreeView />
    </MemoryRouter>,
  )
}

describe('TreeView explorer refresh button', () => {
  beforeEach(() => {
    triggerResyncMock.mockReset().mockResolvedValue(undefined)
    getResyncStatusMock
      .mockReset()
      .mockResolvedValue({ running: false, phase: null, done: true })
    toastMock.success.mockReset()
    toastMock.error.mockReset()
    toastMock.info.mockReset()

    useTreeStore.setState({
      tree: rootNode,
      loading: false,
      error: null,
      pinnedPages: [],
      reloadTree: vi.fn().mockResolvedValue(undefined),
    })
    useSessionStore.setState({ user: null })
    useResyncStore.setState({ isLoading: false, phase: null })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('reloads the tree without resync for non-admins', async () => {
    useSessionStore.setState({
      user: {
        id: '1',
        username: 'editor',
        email: 'editor@example.com',
        role: 'editor',
      },
    })
    renderTreeView()

    fireEvent.click(screen.getByTestId('tree-view-action-button-refresh'))

    await waitFor(() =>
      expect(useTreeStore.getState().reloadTree).toHaveBeenCalledTimes(1),
    )
    expect(triggerResyncMock).not.toHaveBeenCalled()
  })

  it('resyncs and reloads the tree for admins', async () => {
    useSessionStore.setState({
      user: {
        id: '1',
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      },
    })
    renderTreeView()

    fireEvent.click(screen.getByTestId('tree-view-action-button-refresh'))

    await waitFor(() =>
      expect(toastMock.success).toHaveBeenCalledWith('toolbar.refreshSuccess'),
    )
    expect(triggerResyncMock).toHaveBeenCalledTimes(1)
  })

  it('shows an info toast when a resync is already running', async () => {
    useSessionStore.setState({
      user: {
        id: '1',
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      },
    })
    triggerResyncMock.mockRejectedValue(
      new ApiLocalizedError({
        code: 'resync_already_running',
        message: 'Sync already in progress',
        template: '',
      }),
    )
    renderTreeView()

    fireEvent.click(screen.getByTestId('tree-view-action-button-refresh'))

    await waitFor(() =>
      expect(toastMock.info).toHaveBeenCalledWith(
        'toolbar.refreshAlreadyRunning',
      ),
    )
    expect(toastMock.success).not.toHaveBeenCalled()
  })

  it('shows an error toast instead of success when the final tree reload fails', async () => {
    // reloadTree() never rejects — it records failures in useTreeStore's
    // error state instead (see the FIXME in features/tree/state/tree.ts) — so this must
    // be detected by reading that state back, not by awaiting a throw.
    useTreeStore.setState({
      reloadTree: vi.fn().mockImplementation(async () => {
        useTreeStore.setState({ error: 'network error' })
      }),
    })
    useSessionStore.setState({
      user: {
        id: '1',
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      },
    })
    renderTreeView()

    fireEvent.click(screen.getByTestId('tree-view-action-button-refresh'))

    await waitFor(() =>
      expect(toastMock.error).toHaveBeenCalledWith('toolbar.refreshError'),
    )
    expect(toastMock.success).not.toHaveBeenCalled()
  })
})
