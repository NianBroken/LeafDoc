// 数学渲染依赖测试检查锁文件版本，并从各消费包的解析位置加载其实际 KaTeX 副本。
// 安全断言针对继承的 trust 属性，不表示覆盖所有数学输入或全部渲染选项。
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { describe, expect, it } from 'vitest'
const require = createRequire(import.meta.url)
// 只描述本测试调用的同步字符串渲染入口，不引入完整依赖类型。
type Renderer = {
  renderToString: (input: string, options?: Record<string, unknown>) => string
}
describe('KaTeX dependency security', () => {
  it('locks all direct and transitive renderers to patched versions', () => {
    // 枚举所有 katex 包路径并排除类型包，逐份核对允许的版本下界。
    const lock = JSON.parse(readFileSync('package-lock.json', 'utf8')) as {
      packages: Record<string, { version?: string }>
    }
    const copies = Object.entries(lock.packages).filter(
      ([path]) => path.endsWith('/katex') && !path.endsWith('/@types/katex'),
    )
    expect(copies.length).toBeGreaterThan(0)
    for (const [path, entry] of copies) {
      const [major, minor, patch] = entry.version!.split('.').map(Number)
      expect(
        major > 0 || minor > 18 || (minor === 18 && patch >= 2),
        path,
      ).toBe(true)
    }
  })
  for (const consumer of [
    'rehype-katex',
    'micromark-extension-math',
    'mermaid',
  ]) {
    it(`${consumer} ignores inherited trust and still renders ordinary math`, () => {
      // trust 放在原型上而非自有属性，危险链接不能进入 HTML，普通公式仍需正常生成。
      const renderer = createRequire(require.resolve(consumer))(
        'katex',
      ) as Renderer
      const options = Object.assign(Object.create({ trust: true }), {
        throwOnError: false,
        strict: 'ignore',
      }) as Record<string, unknown>
      const html = renderer.renderToString(
        String.raw`\href{javascript:alert(1)}{x}`,
        options,
      )
      expect(html).not.toContain('href="javascript:')
      expect(renderer.renderToString('x^2')).toContain('class="katex"')
    })
  }
})
