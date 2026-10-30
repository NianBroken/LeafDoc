import '@/shared/lib/i18n'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { toast } from 'sonner'
import ForgotPasswordForm from '@/features/auth/ForgotPasswordForm'

vi.mock('sonner', () => ({ toast: { error: vi.fn() } }))

const requestPasswordResetMock = vi.fn()

vi.mock('@/shared/lib/api/auth', () => ({
  requestPasswordReset: (...args: unknown[]) =>
    requestPasswordResetMock(...args),
}))

function renderForm() {
  return render(
    <MemoryRouter initialEntries={['/forgot-password']}>
      <Routes>
        <Route path="/forgot-password" element={<ForgotPasswordForm />} />
        <Route path="/login" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ForgotPasswordForm', () => {
  beforeEach(() => {
    requestPasswordResetMock.mockReset()
  })

  it('shows the same generic confirmation for a successful request', async () => {
    requestPasswordResetMock.mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderForm()

    await user.type(screen.getByTestId('forgot-password-identifier'), 'alice')
    await user.click(screen.getByTestId('forgot-password-submit'))

    await waitFor(() =>
      expect(screen.getByText('Check your email')).toBeInTheDocument(),
    )
    expect(requestPasswordResetMock).toHaveBeenCalledWith('alice')
  })

  // Rejected transport requests remain retryable and never confirm an email was sent.
  it('keeps the form and shows the actual request failure', async () => {
    requestPasswordResetMock.mockRejectedValue(new Error('boom'))
    const user = userEvent.setup()
    renderForm()

    await user.type(
      screen.getByTestId('forgot-password-identifier'),
      'no-such-user',
    )
    await user.click(screen.getByTestId('forgot-password-submit'))

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith('boom'))
  })
})
