// 根据目录项数和全局模式计算每次文档访问时的初始折叠状态，不读取当前面板状态。
export function isTocInitiallyCollapsed(
  entryCount: number,
  mode: string,
): boolean {
  // 空目录与 never 模式折叠；auto 仅在条目不超过三项时折叠，其他模式保持展开。
  return (
    entryCount === 0 || mode === 'never' || (mode === 'auto' && entryCount <= 3)
  )
}
