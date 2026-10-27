import '@/shared/lib/i18n'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import SettingsNav from '@/features/settings/SettingsNav'

const mockUseSettingsSectionContext = vi.fn()

vi.mock('@/shared/lib/registries/settingsSections', async (importOriginal) => {
  const actual =
    await importOriginal<
      typeof import('@/shared/lib/registries/settingsSections')
    >()
  return {
    ...actual,
    useSettingsSectionContext: () => mockUseSettingsSectionContext(),
  }
})

describe('SettingsNav', () => {
  it('shows only the sections visible to a non-admin user', () => {
    mockUseSettingsSectionContext.mockReturnValue({
      role: 'viewer',
      authDisabled: false,
      snapshotEnabled: true,
      enableApiKeyManagement: true,
      userManagementUrl: undefined,
    })

    render(
      <MemoryRouter initialEntries={['/settings/account']}>
        <SettingsNav />
      </MemoryRouter>,
    )

    expect(screen.getByTestId('settings-nav-item-account')).toBeInTheDocument()
    expect(
      screen.queryByTestId('settings-nav-item-branding'),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByTestId('settings-nav-item-backup'),
    ).not.toBeInTheDocument()
  })

  it('shows all enabled sections for an admin, and renders users as an external link when userManagementUrl is set', () => {
    mockUseSettingsSectionContext.mockReturnValue({
      role: 'admin',
      authDisabled: false,
      snapshotEnabled: false,
      enableApiKeyManagement: true,
      userManagementUrl: 'https://control-plane.example.com/users',
    })

    render(
      <MemoryRouter initialEntries={['/settings/account']}>
        <SettingsNav />
      </MemoryRouter>,
    )

    expect(screen.getByTestId('settings-nav-item-branding')).toBeInTheDocument()
    expect(
      screen.queryByTestId('settings-nav-item-snapshots'),
    ).not.toBeInTheDocument()

    const usersLink = screen.getByTestId('settings-nav-item-users')
    expect(usersLink.tagName).toBe('A')
    expect(usersLink).toHaveAttribute(
      'href',
      'https://control-plane.example.com/users',
    )
    expect(usersLink).toHaveAttribute('target', '_blank')
  })

  it('navigates to a section when its nav item is clicked', async () => {
    const user = userEvent.setup({ delay: null })
    mockUseSettingsSectionContext.mockReturnValue({
      role: 'admin',
      authDisabled: false,
      snapshotEnabled: false,
      enableApiKeyManagement: false,
      userManagementUrl: undefined,
    })

    render(
      <MemoryRouter initialEntries={['/settings/account']}>
        <SettingsNav />
      </MemoryRouter>,
    )

    await user.click(screen.getByTestId('settings-nav-item-branding'))

    expect(screen.getByTestId('settings-nav-item-branding')).toHaveClass(
      'list-view__item--active',
    )
  })
})
