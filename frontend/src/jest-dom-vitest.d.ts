// 本声明把 DOM 断言方法接入 Vitest 的两参数 Assertion 接口及非对称匹配接口。
// 类型参数顺序与当前接口一致，只提供编译期类型，实际匹配器由测试设置入口加载。
import type { TestingLibraryMatchers } from '@testing-library/jest-dom/matchers'

// 下方工具指令允许用于接口合并的空接口与保留签名的类型参数，指令文本供 ESLint 识别。
/* eslint-disable @typescript-eslint/no-empty-object-type, @typescript-eslint/no-unused-vars */
declare module 'vitest' {
  interface Assertion<
    R extends void | Promise<void> = void,
    T = unknown,
  > extends TestingLibraryMatchers<unknown, T> {}
  interface AsymmetricMatchersContaining extends TestingLibraryMatchers<
    unknown,
    unknown
  > {}
}
