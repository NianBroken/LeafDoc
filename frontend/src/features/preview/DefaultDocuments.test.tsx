// 真实内置正文通过生产预览组件验证，页面路径来自后端唯一课程定义。
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
const definition = readFileSync(path.join(directory, '../documents.go'), 'utf8')
const documents = Array.from(
  definition.matchAll(
    /\{Slug: "([^"]+)", Title: "([^"]+)", ParentSlug: "([^"]*)"\}/g,
  ),
  ([, slug, title, parent]) => ({ slug, title, parent }),
)
const documentPaths = new Set(
  documents.map(({ slug, parent }) => `/${parent ? `${parent}/` : ''}${slug}`),
)

describe('default document rendering', () => {
  beforeEach(() => {
    const pages = documents.map(({ slug, title, parent }): PageNode => {
      return {
        id: slug,
        title,
        slug,
        path: `${parent ? `${parent}/` : ''}${slug}`,
        version: '1',
        children: null,
        kind: parent ? 'page' : 'section',
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

  it('contains ten substantial documents with three roots and seven children', () => {
    expect(files).toHaveLength(10)
    expect(documents.filter(({ parent }) => !parent)).toHaveLength(3)
    expect(documents.filter(({ parent }) => parent)).toHaveLength(7)
    expect(files.sort()).toEqual(
      documents.map(({ slug }) => `${slug}.md`).sort(),
    )
    const bodies = files.map((file) =>
      readFileSync(`${directory}/${file}`, 'utf8'),
    )
    expect(new Set(bodies).size).toBe(10)
    for (const body of bodies) {
      expect(body.match(/\p{Script=Han}/gu)!.length).toBeGreaterThanOrEqual(700)
    }
  })

  for (const file of files) {
    it(`renders ${file} with valid document structure and working navigation`, () => {
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
      const internalTargets = new Set<string>()
      for (const anchor of container.querySelectorAll('a[href]')) {
        const href = anchor.getAttribute('href')!
        if (href.startsWith('/')) {
          expect(documentPaths.has(href), href).toBe(true)
          internalTargets.add(href)
        }
      }
      expect(internalTargets).toEqual(documentPaths)
      if (file === 'formatting.md') {
        expect(container.querySelector('strong')).not.toBeNull()
        expect(container.querySelector('em')).not.toBeNull()
        expect(container.querySelector('del')).not.toBeNull()
      }
      if (file === 'extensions.md') {
        expect(container.querySelectorAll('.katex-display')).toHaveLength(1)
        expect(content).toContain('```mermaid\nflowchart LR')
      }
      if (file === 'links-assets.md') {
        expect(container.querySelector('img')?.getAttribute('src')).toContain(
          '/defaults/leafdoc-logo.webp',
        )
      }
      if (file === 'tables.md') {
        expect(container.querySelectorAll('table thead th')).toHaveLength(3)
        expect(container.querySelectorAll('table tbody tr')).toHaveLength(2)
      }
      if (file === 'lists.md') {
        expect(
          container.querySelectorAll('input[type="checkbox"]'),
        ).toHaveLength(2)
        expect(container.querySelectorAll('input[checked]')).toHaveLength(1)
      }
    })
  }
})
