import { readFileSync, readdirSync } from 'node:fs'
import path from 'node:path'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { TooltipProvider } from '@/shared/components/ui/tooltip'
import MarkdownPreview from './MarkdownPreview'
import { useTreeStore } from '@/features/tree/state/tree'
import type { PageNode } from '@/shared/lib/api/pages'

const directory = path.resolve(
  process.cwd(),
  '../backend/internal/core/defaultdocs/content',
)
const files = readdirSync(directory).filter((file) => file.endsWith('.md'))

// Validate the real embedded curriculum with the production Markdown pipeline.
describe('default document rendering', () => {
  beforeEach(() => {
    const pages = files.map((file): PageNode => {
      const slug = file.replace(/\.md$/, '')
      return {
        id: slug,
        title: slug,
        slug,
        path: slug,
        version: '1',
        children: null,
        kind: 'page',
      }
    })
    useTreeStore.setState({
      byPath: Object.fromEntries(pages.map((page) => [page.path, page])),
    })
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
        1000,
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
      expect(container.querySelectorAll('pre code').length).toBeGreaterThan(0)
      const documentPaths = new Set(
        files.map((name) => `/${name.replace(/\.md$/, '')}`),
      )
      const internalTargets = new Set<string>()
      for (const anchor of container.querySelectorAll('a[href]')) {
        const href = anchor.getAttribute('href')!
        if (href.startsWith('#')) {
          expect(container.querySelector(href), href).not.toBeNull()
        } else if (href.startsWith('/')) {
          expect(documentPaths.has(href), href).toBe(true)
          internalTargets.add(href)
        } else {
          expect(['spec.commonmark.org', 'github.github.com']).toContain(
            new URL(href).hostname,
          )
          expect(new URL(href).protocol).toBe('https:')
        }
      }
      expect(internalTargets).toEqual(documentPaths)
      if (file.includes('text-and-code')) {
        expect(container.querySelector('strong')?.textContent).toBe(
          '必须保存的设置',
        )
        expect(container.querySelector('em')?.textContent).toBe('辅助说明')
        expect(container.querySelector('del')?.textContent).toBe('旧入口')
        expect(container.querySelector('p code')?.textContent).toBe('guide.md')
      }
      if (file.includes('extensions-and-troubleshooting')) {
        expect(container.querySelectorAll('.katex')).toHaveLength(2)
        expect(container.querySelectorAll('.katex-display')).toHaveLength(1)
        expect(content).toContain('```mermaid\nflowchart TD')
      }
      if (file.includes('links-images')) {
        expect(container.querySelectorAll('table')).toHaveLength(1)
        expect(container.querySelectorAll('table thead th')).toHaveLength(3)
        expect(container.querySelectorAll('table tbody tr')).toHaveLength(3)
        expect(container.querySelector('img')?.getAttribute('src')).toContain(
          '/defaults/leafdoc-logo.webp',
        )
        expect(container.querySelector('#link-check')).not.toBeNull()
      }
      if (file.includes('lists-and-structure')) {
        expect(
          container.querySelectorAll('input[type="checkbox"]'),
        ).toHaveLength(4)
        expect(container.querySelectorAll('input[checked]')).toHaveLength(2)
        expect(container.querySelectorAll('blockquote').length).toBeGreaterThan(
          0,
        )
      }
    })
  }
})
