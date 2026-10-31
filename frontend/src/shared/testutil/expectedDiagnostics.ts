// 本文件核对测试中明确预期的控制台诊断，供失败分支用例检查日志级别、顺序和原始参数。
// 设置诊断文件环境变量时，把已经验证的调用参数写成独立 JSON 行，便于检查容器测试记录。
import { appendFileSync } from 'node:fs'
import { inspect } from 'node:util'
import { expect, onTestFinished, vi } from 'vitest'

/**
 * 监听指定级别的控制台调用，每次调用都必须与 expected 的同一下标参数数组一致。
 * 测试结束时核对总次数，并在断言成功或失败后都恢复原方法，避免影响其他测试。
 * 调用方应在触发操作前注册监听；本入口不接受未声明的额外日志，也不匹配其他级别。
 */
export function expectDiagnostics(
  level: 'warn' | 'error',
  expected: unknown[][],
) {
  let received = 0 // 仅在当前调用的参数断言通过后递增，结束时与预期条数比较。
  const spy = vi
    .spyOn(console, level)
    .mockImplementation((...args: unknown[]) => {
      expect(args).toEqual(expected[received])
      received++
      // 未设置路径时只断言，不创建文件；设置路径时保留完整参数的可读形式。
      const recordPath = process.env.LEAFDOC_TEST_DIAGNOSTICS
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
  // 恢复操作放在 finally 中，次数不符也不能让控制台继续处于测试替身状态。
  onTestFinished(() => {
    try {
      expect(received).toBe(expected.length)
    } finally {
      spy.mockRestore()
    }
  })
}
