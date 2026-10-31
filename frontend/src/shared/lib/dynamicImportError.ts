// 三种浏览器动态模块加载失败提示的文本模式，按大小写不敏感方式识别。
// 文本匹配只分类提示，不确认 HTTP 状态、部署版本或故障能否通过刷新解决。
const DYNAMIC_IMPORT_ERROR_PATTERNS = [
  /failed to fetch dynamically imported module/i,
  /error loading dynamically imported module/i,
  /importing a module script failed/i,
]

// 任一模式命中即返回 true，不在本函数重试导入或刷新页面。
export function isDynamicImportChunkError(message: string): boolean {
  return DYNAMIC_IMPORT_ERROR_PATTERNS.some((pattern) => pattern.test(message))
}
