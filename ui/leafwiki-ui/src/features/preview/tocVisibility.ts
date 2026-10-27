// The global mode sets only the initial visibility for each document visit.
export function isTocInitiallyCollapsed(
  entryCount: number,
  mode: string,
): boolean {
  return (
    entryCount === 0 || mode === 'never' || (mode === 'auto' && entryCount <= 3)
  )
}
