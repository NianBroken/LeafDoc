// 本配置组合 TypeScript、React 和源码安全检查，规则只作用于各块匹配的路径。
import js from '@eslint/js'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import globals from 'globals'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  // 这些路径整体跳过检查，组件库目录不会进入后续 src 规则块。
  { ignores: ['dist', 'src/shared/components/ui', 'node_modules'] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'no-useless-assignment': 'off',
      'preserve-caught-error': 'off',
      'react-hooks/set-state-in-effect': 'off',
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
    },
  },
  {
    // 业务源码禁止所列动态执行和直接写入 HTML 的语法，测试文件与测试初始化除外。
    // 选择器检查语法形式，不分析字符串是否已净化，也不追踪别名或计算属性中的调用。
    // 确需逐行豁免时，工具控制标记保留原文，并在附近说明输入来源及安全条件。
    files: ['src/**/*.{ts,tsx}'],
    ignores: ['**/*.test.{ts,tsx}', 'src/test-setup.ts'],
    rules: {
      'no-eval': 'error',
      'no-implied-eval': 'error',
      'no-new-func': 'error',
      'no-restricted-syntax': [
        'error',
        {
          selector: "JSXAttribute[name.name='dangerouslySetInnerHTML']",
          message:
            'dangerouslySetInnerHTML bypasses sanitization. Render through the Markdown pipeline or React elements.',
        },
        {
          selector: "JSXAttribute[name.name='srcDoc']",
          message: 'srcDoc injects raw HTML into an iframe.',
        },
        {
          selector:
            'AssignmentExpression[left.type="MemberExpression"][left.property.name=/^(innerHTML|outerHTML)$/]',
          message:
            'Assigning innerHTML/outerHTML bypasses sanitization. Use textContent or React elements.',
        },
        {
          selector: "CallExpression[callee.property.name='insertAdjacentHTML']",
          message: 'insertAdjacentHTML bypasses sanitization.',
        },
        {
          selector:
            "CallExpression[callee.object.name='document'][callee.property.name=/^(write|writeln)$/]",
          message: 'document.write bypasses sanitization.',
        },
      ],
    },
  },
)
