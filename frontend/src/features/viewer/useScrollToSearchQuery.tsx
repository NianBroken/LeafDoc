import { scrollToSearchQuery } from '@/shared/lib/scrollToSearchQuery'
import { useEffect } from 'react'
import { useLocation, useSearchParams } from 'react-router'

type UseScrollToSearchQueryOptions = {
  content?: string
  isLoading?: boolean
}

export function useScrollToSearchQuery({
  content,
  isLoading,
}: UseScrollToSearchQueryOptions) {
  const { hash } = useLocation()
  const [searchParams] = useSearchParams()
  const query = (searchParams.get('q') ?? '').trim()

  useEffect(() => {
    // 同时存在搜索词和标题锚点时优先定位标题锚点。
    if (isLoading || !content || hash || query.length < 3) return
    return scrollToSearchQuery(query)
  }, [content, isLoading, hash, query])
}
