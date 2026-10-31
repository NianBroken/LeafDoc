/// <reference types="vite/client" />

// 上方三斜线指令加载 Vite 客户端类型，保持其机器识别语法和文件起始位置。
// 字体包只作样式副作用导入，此空模块声明允许 TypeScript 解析该导入。
declare module '@fontsource/inter'
// GFM 插件的五个导出均接收 Turndown 实例并返回空值，这里只声明调用形状。
// 声明文件不执行转换、不注册插件，实际使用入口负责调用所需导出。
declare module 'turndown-plugin-gfm' {
  import TurndownService from 'turndown'
  export function gfm(service: TurndownService): void
  export function tables(service: TurndownService): void
  export function strikethrough(service: TurndownService): void
  export function taskListItems(service: TurndownService): void
  export function highlightedCodeBlock(service: TurndownService): void
}
