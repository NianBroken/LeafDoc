import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { beforeEach, expect, it, vi } from 'vitest'
import { login } from '@/shared/lib/api/auth'
import { useSessionStore } from '@/shared/state/session'
import LoginForm from '@/features/auth/LoginForm'
vi.mock('@/shared/lib/api/auth', () => ({ login: vi.fn() }))
vi.mock('sonner', () => ({ toast: { error: vi.fn() } }))
beforeEach(() => {
  useSessionStore.setState({ user: null })
  vi.clearAllMocks()
})
it('shows a password login form', () => {
  render(
    <MemoryRouter>
      <LoginForm />
    </MemoryRouter>,
  )
  expect(screen.getByTestId('login-password')).toBeInTheDocument()
})
it('submits the entered credentials', async () => {
  vi.mocked(login).mockRejectedValue(new Error('test'))
  render(
    <MemoryRouter>
      <LoginForm />
    </MemoryRouter>,
  )
  await userEvent.type(screen.getByTestId('login-identifier'), 'NianBroken')
  await userEvent.type(screen.getByTestId('login-password'), 'Nian123456')
  await userEvent.click(screen.getByTestId('login-submit'))
  expect(login).toHaveBeenCalledWith('NianBroken', 'Nian123456')
})
