import { useBrandingStore } from '@/features/branding/state/branding'
import { useEffect } from 'react'

export interface UseSetTitleOptions {
  title: string
}

// 标题或站点名变化后写入 document.title，非空标题以半角连字符连接站点名。
// 空标题只显示站点名，卸载时不恢复进入页面前的浏览器标题。
export function useSetTitle({ title }: UseSetTitleOptions) {
  const siteNameFromStore = useBrandingStore((s) => s.siteName)

  useEffect(() => {
    document.title = title
      ? `${title} - ${siteNameFromStore}`
      : siteNameFromStore
  }, [title, siteNameFromStore])
}
