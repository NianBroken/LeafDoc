import { useBrandingStore } from '@/features/branding/state/branding'
import { useEffect } from 'react'

export interface UseSetTitleOptions {
  title: string
}

// 将页面标题同步到浏览器文档标题。
export function useSetTitle({ title }: UseSetTitleOptions) {
  const siteNameFromStore = useBrandingStore((s) => s.siteName)

  useEffect(() => {
    document.title = title
      ? `${title} - ${siteNameFromStore}`
      : siteNameFromStore
  }, [title, siteNameFromStore])
}
