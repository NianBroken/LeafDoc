import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router'
import Importer from './Importer'
import { useImportStore } from './state/import'

const initialState = useImportStore.getState()

afterEach(() => {
  cleanup()
  useImportStore.setState(initialState, true)
})

// 执行失败且没有结果数据时，页面仍显示可用于恢复操作的入口。
describe('importer execution failure', () => {
  it('shows the plan, failure reason and new-import action without result items', () => {
    useImportStore.setState({
      loadImportPlan: vi.fn().mockResolvedValue(undefined),
      importResult: null,
      importPlan: {
        id: 'failed-plan',
        tree_hash: 'before',
        execution_status: 'failed',
        execution_error: 'plan is stale: expected an unchanged document tree',
        cancel_requested: false,
        processed_items: 0,
        total_items: 0,
        errors: [],
        items: [],
      },
    })
    render(
      <MemoryRouter>
        <Importer />
      </MemoryRouter>,
    )
    expect(
      screen.getByText('plan is stale: expected an unchanged document tree'),
    ).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'Start New Import' }),
    ).toBeEnabled()
    expect(
      screen.queryByRole('button', { name: 'Execute Import Plan' }),
    ).not.toBeInTheDocument()
  })
})
