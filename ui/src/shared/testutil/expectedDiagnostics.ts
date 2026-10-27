import { appendFileSync } from 'node:fs'
import { inspect } from 'node:util'
import { expect, onTestFinished, vi } from 'vitest'

/** Verifies every declared diagnostic in order and retains its original content. */
export function expectDiagnostics(
  level: 'warn' | 'error',
  expected: unknown[][],
) {
  let received = 0
  const spy = vi
    .spyOn(console, level)
    .mockImplementation((...args: unknown[]) => {
      expect(args).toEqual(expected[received])
      received++
      const recordPath = process.env.LEAFWIKI_TEST_DIAGNOSTICS
      if (recordPath) {
        appendFileSync(
          recordPath,
          JSON.stringify({
            test: expect.getState().currentTestName,
            level,
            diagnostic: inspect(args, { depth: null, colors: false }),
          }) + '\n',
          'utf8',
        )
      }
    })
  onTestFinished(() => {
    try {
      expect(received).toBe(expected.length)
    } finally {
      spy.mockRestore()
    }
  })
}
