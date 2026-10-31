// 构建文件名测试检查下划线与点前缀被去除，普通分块名称保持原样。
import { describe, expect, it } from 'vitest'
import { chunkFileName } from './vite.config'

describe('chunkFileName', () => {
  it('never emits a file name starting with "_" or "."', () => {
    // 目录形式的 go:embed 会跳过以下划线或点开头的文件，分块名必须避开这两个前缀。
    // 断言直接比较命名结果，实际嵌入与浏览器加载不由这两个用例执行。
    expect(chunkFileName({ name: '_baseUniq' })).toBe(
      'static/baseUniq-[hash].js',
    )
    expect(chunkFileName({ name: '.hidden' })).toBe('static/hidden-[hash].js')
  })

  // 普通名称只套用静态目录和散列占位符，不改写名称正文。
  it('leaves regular chunk names untouched', () => {
    expect(chunkFileName({ name: 'codemirror' })).toBe(
      'static/codemirror-[hash].js',
    )
  })
})
