import '@/shared/lib/i18n'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { BrandingAsset } from './BrandingAsset'

// Exercise real image error events and URL changes without browser networking.
describe('BrandingAsset', () => {
  it('shows the built-in logo when no custom file exists', () => {
    render(
      <BrandingAsset kind="logo" file="" version={1} alt="Logo" showSource />,
    )
    expect(screen.getByRole('img', { name: 'Logo' })).toHaveTextContent('🌿')
    expect(screen.getByText('Built-in logo')).toBeInTheDocument()
  })

  it('falls back from a broken logo and retries when its version changes', () => {
    const { rerender } = render(
      <BrandingAsset kind="logo" file="logo.png" version={1} alt="Logo" />,
    )
    fireEvent.error(screen.getByRole('img'))
    expect(screen.getByRole('img')).toHaveTextContent('🌿')
    rerender(
      <BrandingAsset kind="logo" file="logo.png" version={2} alt="Logo" />,
    )
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      '/branding/logo.png?v=2',
    )
  })

  it('previews the favicon actually used after custom deletion', () => {
    const { rerender } = render(
      <BrandingAsset
        kind="favicon"
        file="favicon.png"
        version={1}
        alt="Icon"
      />,
    )
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      '/branding/favicon.png?v=1',
    )
    rerender(
      <BrandingAsset
        kind="favicon"
        file=""
        version={2}
        alt="Icon"
        showSource
      />,
    )
    expect(screen.getByRole('img')).toHaveAttribute('src', '/favicon.svg?v=2')
    expect(screen.getByText('Built-in favicon')).toBeInTheDocument()
  })

  it('reports absence only after both favicon sources fail', () => {
    render(
      <BrandingAsset
        kind="favicon"
        file="favicon.png"
        version={1}
        alt="Icon"
      />,
    )
    fireEvent.error(screen.getByRole('img'))
    expect(screen.getByRole('img')).toHaveAttribute('src', '/favicon.svg?v=1')
    fireEvent.error(screen.getByRole('img'))
    expect(screen.getByRole('status')).toHaveTextContent('No available favicon')
  })
})
