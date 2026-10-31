// 前端测试使用 React 转换与 jsdom，测试路径和覆盖率范围分别配置。
import react from '@vitejs/plugin-react'
import path from 'path'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react()],
  test: {
    // 预载公共设置并启用全局测试函数，同时纳入 src 测试和 Vite 配置测试。
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    include: ['src/**/*.test.{ts,tsx}', 'vite.config.test.ts'],
    coverage: {
      // 覆盖率使用 V8，排除测试本身、设置入口和应用主入口，生成终端与 lcov 报告。
      provider: 'v8',
      reporter: ['text', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/test-setup.ts', 'src/main.tsx'],
    },
  },
  resolve: {
    // 测试与应用共用 @ 指向 src 的绝对位置，路径根据配置文件目录计算。
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
})
