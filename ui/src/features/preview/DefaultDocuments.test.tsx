import { readFileSync, readdirSync } from 'node:fs'
import path from 'node:path'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { TooltipProvider } from '@/shared/components/ui/tooltip'
import MarkdownPreview from './MarkdownPreview'

const directory = path.resolve(
  process.cwd(),
  '../internal/core/defaultdocs/content',
)
const files = readdirSync(directory).filter((file) => file.endsWith('.md'))

// Validate the real embedded curriculum with the production Markdown pipeline.
describe('default document rendering', () => {
  beforeEach(() => {
    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }))
  })
  it('contains five distinct substantial documents', () => {
    expect(files).toHaveLength(5)
    const bodies = files.map((file) =>
      readFileSync(`${directory}/${file}`, 'utf8'),
    )
    expect(new Set(bodies).size).toBe(5)
    for (const body of bodies) {
      expect(body.match(/\p{Script=Han}/gu)!.length).toBeGreaterThanOrEqual(
        2000,
      )
    }
  })

  for (const file of files) {
    it(`renders ${file} with valid document structure`, () => {
      const content = readFileSync(`${directory}/${file}`, 'utf8')
      const { container } = render(
        <MemoryRouter>
          <TooltipProvider>
            <MarkdownPreview content={content} />
          </TooltipProvider>
        </MemoryRouter>,
      )
      expect(container.querySelectorAll('h1')).toHaveLength(1)
      expect(container.querySelectorAll('h2').length).toBeGreaterThan(5)
      expect(container.querySelector('script')).toBeNull()
      if (file.includes('text-and-code')) {
        expect(container.querySelector('sub')?.textContent).toBe('2')
        expect(container.querySelector('sup')?.textContent).toBe('2')
        expect(container.querySelectorAll('.katex').length).toBeGreaterThan(0)
      }
      if (file.includes('links-images')) {
        expect(container.querySelectorAll('table')).toHaveLength(2)
        expect(container.querySelector('img')?.getAttribute('src')).toContain(
          '/defaults/leafdoc-logo.webp',
        )
        expect(container.querySelector('#表格基本写法')).not.toBeNull()
      }
      if (file.includes('lists-and-structure')) {
        expect(
          container.querySelectorAll('input[type="checkbox"]'),
        ).toHaveLength(3)
        expect(container.querySelectorAll('blockquote').length).toBeGreaterThan(
          0,
        )
      }
    })
  }
})
