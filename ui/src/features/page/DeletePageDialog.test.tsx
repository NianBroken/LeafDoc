import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { DeletePageDialog } from './DeletePageDialog'
import { useTreeStore } from '@/features/tree/state/tree'
import { useViewerStore } from '@/features/viewer/viewer'
import { useConfigStore } from '@/shared/state/config'
import { useDialogsStore } from '@/shared/state/dialogs'
import { DIALOG_DELETE_PAGE_CONFIRMATION } from '@/shared/lib/registries/instances'
import { deletePage, type PageNode } from '@/shared/lib/api/pages'
import { fetchLinkStatus, type LinkStatusResult } from '@/shared/lib/api/links'

vi.mock('@/shared/lib/api/links', () => ({ fetchLinkStatus: vi.fn() }))
vi.mock('@/shared/lib/api/pages', async () => ({
  ...(await vi.importActual('@/shared/lib/api/pages')),
  deletePage: vi.fn(),
}))

const page: PageNode = {
  id: 'delete-target',
  title: 'Delete target',
  slug: 'delete-target',
  path: 'delete-target',
  version: '1',
  kind: 'page',
  children: null,
}
const links: LinkStatusResult = {
  backlinks: [],
  broken_incoming: [],
  outgoings: [],
  broken_outgoings: [],
  counts: {
    backlinks: 0,
    broken_incoming: 0,
    outgoings: 0,
    broken_outgoings: 0,
  },
}

// A pending reference request must finish before its target can disappear.
describe('DeletePageDialog reference lifecycle', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useTreeStore.setState({
      byId: { [page.id]: page },
      reloadTree: vi.fn().mockResolvedValue(undefined),
    })
    useConfigStore.setState({ enableLinkRefactor: true })
    useDialogsStore.setState({
      dialogType: DIALOG_DELETE_PAGE_CONFIRMATION,
      dialogProps: null,
    })
    useViewerStore.setState({ page: { ...page, content: 'Current page' } })
    vi.mocked(deletePage).mockResolvedValue(undefined)
  })

  it('blocks confirmation until references finish and clears the deleted viewer', async () => {
    let completeReferences!: (result: LinkStatusResult) => void
    vi.mocked(fetchLinkStatus).mockReturnValue(
      new Promise((resolve) => {
        completeReferences = resolve
      }),
    )
    render(
      <MemoryRouter>
        <DeletePageDialog pageId={page.id} redirectTo="/" />
      </MemoryRouter>,
    )
    const confirm = screen.getByTestId('delete-page-dialog-button-confirm')
    expect(confirm).toBeDisabled()
    await userEvent.click(confirm)
    expect(deletePage).not.toHaveBeenCalled()
    await act(async () => {
      completeReferences(links)
    })
    await waitFor(() => expect(confirm).toBeEnabled())
    await userEvent.click(confirm)
    await waitFor(() =>
      expect(deletePage).toHaveBeenCalledWith(page.id, false, page.version),
    )
    expect(useViewerStore.getState().page).toBeNull()
  })
})
