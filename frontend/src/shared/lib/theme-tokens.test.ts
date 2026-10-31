/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * 按 CSS 文本中的出现顺序比较 --muted-surface 与 --muted-foreground 的声明值。
 * 两组声明必须数量相同且对应文本不同，避免表面颜色与前景颜色使用同一声明。
 * 测试不解析选择器或计算颜色，不证明两种不同写法的颜色在视觉上具有足够对比度。
 * 文件顶部的 reference 是 TypeScript 类型指令，保留其原始语法供工具读取。
 */
// 测试从前端包根目录执行，读取该目录下的 src/index.css，不依赖测试文件所在目录。
const cssText = readFileSync(resolve(process.cwd(), 'src/index.css'), 'utf8')

// 提取以分号结束的属性值，忽略 !important 并把连续空白归为一个空格。
// 正则比较的是文本声明，不展开 CSS 变量、颜色函数或继承关系。
function valuesOf(prop: 'muted-surface' | 'muted-foreground'): string[] {
  const re = new RegExp(`--${prop}:\\s*([^;!]+?)\\s*(?:!important)?;`, 'g')
  const out: string[] = []
  for (const m of cssText.matchAll(re)) {
    out.push(m[1].trim().replace(/\s+/g, ' '))
  }
  return out
}

describe('theme tokens', () => {
  it('defines --muted-surface separately from --muted-foreground in every theme block', () => {
    // 以两组声明的相同下标配对，失败提示包含声明次序，便于定位对应主题文本。
    const surface = valuesOf('muted-surface')
    const foreground = valuesOf('muted-foreground')

    expect(surface.length).toBeGreaterThan(0)
    expect(surface.length).toBe(foreground.length)

    surface.forEach((value, i) => {
      expect(
        value,
        `--muted-surface and --muted-foreground are identical ("${value}") in theme block #${i + 1}`,
      ).not.toBe(foreground[i])
    })
  })
})
