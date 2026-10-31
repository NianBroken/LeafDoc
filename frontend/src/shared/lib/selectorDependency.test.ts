// 选择器解析依赖测试检查锁文件中的全部副本，并执行长平面选择器的实际解析。
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
describe('selector parser security dependency', () => {
  it('locks every selector parser copy to a patched version', () => {
    // 按安装路径后缀收集依赖，至少存在一份且每份版本都符合断言下界。
    const lock = JSON.parse(readFileSync('package-lock.json', 'utf8')) as {
      packages: Record<string, { version?: string }>
    }
    const copies = Object.entries(lock.packages).filter(([path]) =>
      path.endsWith('/postcss-selector-parser'),
    )
    expect(copies.length).toBeGreaterThan(0)
    for (const [path, entry] of copies) {
      const [major, minor, patch] = entry.version!.split('.').map(Number)
      expect(
        major > 7 ||
          (major === 7 && (minor > 1 || (minor === 1 && patch >= 6))),
        path,
      ).toBe(true)
    }
  })
  it('parses a long flat selector without changing its nodes', () => {
    // 五千个连续类选择器同时约束输出文本与节点数，不用耗时阈值代替解析结果。
    const parser = require('postcss-selector-parser') as (
      callback?: (root: { nodes: { nodes: unknown[] }[] }) => void,
    ) => { processSync: (input: string) => string }
    const input = '.a'.repeat(5000)
    let count = 0
    expect(
      parser((root) => {
        count = root.nodes[0].nodes.length
      }).processSync(input),
    ).toBe(input)
    expect(count).toBe(5000)
  })
})
