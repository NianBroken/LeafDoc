import { describe, expect, it } from 'vitest'
import { chunkFileName } from './vite.config'

describe('chunkFileName', () => {
  it('never emits a file name starting with "_" or "."', () => {
    // Go's //go:embed skips files starting with "_" or "." when embedding a
    // directory, so such a chunk (e.g. mermaid's lodash "_baseUniq") would
    // 404 at runtime and break the lazy import that depends on it.
    expect(chunkFileName({ name: '_baseUniq' })).toBe(
      'static/baseUniq-[hash].js',
    )
    expect(chunkFileName({ name: '.hidden' })).toBe('static/hidden-[hash].js')
  })

  it('leaves regular chunk names untouched', () => {
    expect(chunkFileName({ name: 'codemirror' })).toBe(
      'static/codemirror-[hash].js',
    )
  })
})
