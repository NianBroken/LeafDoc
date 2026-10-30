import { DIALOG_IMAGE_PREVIEW } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MarkdownImage } from '@/features/preview/MarkdownImage'

describe('MarkdownImage', () => {
  beforeEach(() => {
    useDialogsStore.setState({ dialogType: null, dialogProps: null })
  })

  it('appends a ?v= cache-busting param to an /assets/ src', () => {
    render(<MarkdownImage src="/assets/foo.png" alt="foo" />)

    const src = screen.getByAltText('foo').getAttribute('src') ?? ''
    expect(src).toContain('/assets/foo.png')
    expect(src).toMatch(/\?v=\d+$/)
  })

  it.each([undefined, ''])(
    'does not load or preview a missing source %j',
    (src) => {
      const openWindow = vi.spyOn(window, 'open')
      try {
        render(<MarkdownImage src={src} alt="missing" />)
        const image = screen.getByAltText('missing')
        expect(image).not.toHaveAttribute('src')
        fireEvent.click(image)
        fireEvent.click(image, { ctrlKey: true })
        expect(useDialogsStore.getState().dialogType).toBeNull()
        expect(openWindow).not.toHaveBeenCalled()
      } finally {
        openWindow.mockRestore()
      }
    },
  )

  it('opens the image preview dialog when not wrapped in a link', () => {
    render(<MarkdownImage src="/assets/foo.png" alt="foo" />)

    fireEvent.click(screen.getByAltText('foo'))

    expect(useDialogsStore.getState().dialogType).toBe(DIALOG_IMAGE_PREVIEW)
  })

  it('renders inline-block so trailing text stays on the same line (#1471)', () => {
    render(<MarkdownImage src="/assets/foo.png" alt="foo" />)

    expect(screen.getByAltText('foo')).toHaveStyle({ display: 'inline-block' })
  })

  it('has no vertical margin so paragraph spacing governs the gap around it (#1524)', () => {
    render(<MarkdownImage src="/assets/foo.png" alt="foo" />)

    expect(screen.getByAltText('foo')).toHaveStyle({
      marginTop: '0px',
      marginBottom: '0px',
    })
  })

  it('lets the surrounding link handle the click instead of opening the preview', () => {
    // Observe the parent link event without asking JSDOM to perform browser navigation.
    const handleLinkClick = vi.fn((event: React.MouseEvent) =>
      event.preventDefault(),
    )
    render(
      <a href="https://example.com" onClick={handleLinkClick}>
        <MarkdownImage src="/assets/foo.png" alt="foo" />
      </a>,
    )

    fireEvent.click(screen.getByAltText('foo'))

    expect(useDialogsStore.getState().dialogType).toBeNull()
    expect(handleLinkClick).toHaveBeenCalledOnce()
  })
})
