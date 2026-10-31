// 查看器从 q 参数提取搜索词，正文就绪后请求共享函数定位并标记匹配文字。
import { scrollToSearchQuery } from '@/shared/lib/scrollToSearchQuery'
import { useEffect } from 'react'
import { useLocation, useSearchParams } from 'react-router'

type UseScrollToSearchQueryOptions = {
  content?: string
  isLoading?: boolean
}

// 搜索词去除首尾空白且至少包含三个 UTF-16 单元；副作用返回定位函数的清理入口。
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
