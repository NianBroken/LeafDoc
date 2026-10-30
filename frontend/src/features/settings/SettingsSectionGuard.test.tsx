import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import SettingsSectionGuard from '@/features/settings/SettingsSectionGuard'

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

function renderGuard(section: {
  id: string
  roles: string[]
  isEnabled?: (ctx: unknown) => boolean
  externalHref?: (ctx: unknown) => string | undefined
}) {
  return render(
    <MemoryRouter initialEntries={['/settings/target']}>
      <Routes>
        <Route
          path="/settings/target"
          element={
            <SettingsSectionGuard
              section={
                section as unknown as import('@/shared/lib/registries/settingsSections').SettingsSection
              }
            >
              <div data-testid="section-content">content</div>
            </SettingsSectionGuard>
          }
        />
        <Route
          path="/settings"
          element={<div data-testid="settings-index">index</div>}
        />
      </Routes>
    </MemoryRouter>,
  )
}

describe('SettingsSectionGuard', () => {
  it('renders the section when it is visible and not external', () => {
    mockUseSettingsSectionContext.mockReturnValue({ role: 'admin' })

    renderGuard({ id: 'branding', roles: ['admin'] })

    expect(screen.getByTestId('section-content')).toBeInTheDocument()
  })

  it('redirects to /settings when the role does not match', () => {
    mockUseSettingsSectionContext.mockReturnValue({ role: 'viewer' })

    renderGuard({ id: 'branding', roles: ['admin'] })

    expect(screen.getByTestId('settings-index')).toBeInTheDocument()
    expect(screen.queryByTestId('section-content')).not.toBeInTheDocument()
  })

  it('redirects to /settings when isEnabled returns false (the backup/snapshots URL-bypass gap)', () => {
    mockUseSettingsSectionContext.mockReturnValue({
      role: 'admin',
    })

    renderGuard({
      id: 'backup',
      roles: ['admin'],
      isEnabled: (ctx: unknown) =>
        !!(ctx as { gitBackupEnabled?: boolean }).gitBackupEnabled,
    })

    expect(screen.getByTestId('settings-index')).toBeInTheDocument()
    expect(screen.queryByTestId('section-content')).not.toBeInTheDocument()
  })

  it('redirects to /settings for a visible section with an externalHref instead of rendering it', () => {
    mockUseSettingsSectionContext.mockReturnValue({
      role: 'admin',
      userManagementUrl: 'https://control-plane.example.com/users',
    })

    renderGuard({
      id: 'users',
      roles: ['admin'],
      externalHref: (ctx: unknown) =>
        (ctx as { userManagementUrl?: string }).userManagementUrl,
    })

    expect(screen.getByTestId('settings-index')).toBeInTheDocument()
    expect(screen.queryByTestId('section-content')).not.toBeInTheDocument()
  })
})
