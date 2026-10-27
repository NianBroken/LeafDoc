// Session changes update route access without replacing browser history ownership.
import { act, render, screen } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router'
import type { ReactNode } from 'react'
import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import AuthWrapper from './RouterAuthWrapper'
vi.mock('@/app/layout/AppLayout', () => ({
  default: ({ children }: { children: ReactNode }) => children,
}))
beforeEach(() => {
  useConfigStore.setState({
    publicAccess: true,
    authDisabled: false,
    loginUrl: '',
  })
  useSessionStore.setState({
    user: {
      id: 'owner',
      username: 'owner',
      email: 'owner@example.com',
      role: 'admin',
    },
    isRefreshing: false,
  })
})
it('leaves a protected settings route when the session is cleared', () => {
  render(
    <MemoryRouter initialEntries={['/protected']}>
      <Routes>
        <Route
          path="/protected"
          element={
            <AuthWrapper>
              <span>Settings content</span>
            </AuthWrapper>
          }
        />
        <Route path="/" element={<span>Public home</span>} />
      </Routes>
    </MemoryRouter>,
  )
  expect(screen.getByText('Settings content')).toBeVisible()
  act(() => useSessionStore.getState().clear())
  expect(screen.getByText('Public home')).toBeVisible()
})
it('keeps public content accessible when the same session is cleared', () => {
  render(
    <MemoryRouter>
      <AuthWrapper allowPublic>
        <span>Public content</span>
      </AuthWrapper>
    </MemoryRouter>,
  )
  act(() => useSessionStore.getState().clear())
  expect(screen.getByText('Public content')).toBeVisible()
})
