// 有标题时显示页内目录名称，无标题但有下载项时显示下载列表名称，两者都为空仍使用页内目录名称。
export function tocTitleKey(
  entryCount: number,
  downloadCount: number,
): 'toc.onThisPage' | 'toc.downloads' {
  if (entryCount > 0) return 'toc.onThisPage'
  if (downloadCount > 0) return 'toc.downloads'
  return 'toc.onThisPage'
}
