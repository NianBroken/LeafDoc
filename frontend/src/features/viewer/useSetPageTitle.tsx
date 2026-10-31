import { Page } from '@/shared/lib/api/pages'
import { useBrandingStore } from '@/features/branding/state/branding'
import { useEffect } from 'react'

export interface UseSetPageTitleOptions {
  page: Page | null
}

// 页面标题或站点名变化后写入 document.title，页面为空或标题为空时只显示站点名。
// 依赖项只观察 page.title，不因页面其他字段变化重复写标题，卸载时不恢复旧标题。
export function useSetPageTitle({ page }: UseSetPageTitleOptions) {
  const siteName = useBrandingStore((s) => s.siteName)

  useEffect(() => {
    document.title = page?.title ? `${page?.title} - ${siteName}` : siteName
  }, [page?.title, siteName])
}
