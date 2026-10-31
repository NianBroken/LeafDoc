// 本文件只用纯字符串核验行拆分，元素克隆行为不在这组三个断言内。
import { splitHighlightedLines } from '@/features/preview/splitHighlightedLines'

describe('splitHighlightedLines', () => {
  // 每个普通文本行形成独立数组，顺序与输入换行一致。
  it('splits plain text on newlines', () => {
    expect(splitHighlightedLines('a\nb\nc')).toEqual([['a'], ['b'], ['c']])
  })

  // 末尾一个换行产生的空行被移除，避免渲染额外的行号。
  it('drops a trailing empty line from a final newline', () => {
    expect(splitHighlightedLines('a\nb\n')).toEqual([['a'], ['b']])
  })

  // 连续换行产生的中间空行保持为空数组，不与前后正文合并。
  it('keeps a blank middle line', () => {
    expect(splitHighlightedLines('a\n\nb')).toEqual([['a'], [], ['b']])
  })
})
