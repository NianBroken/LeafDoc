// 查看器在正文可用后消费地址片段，具体标题定位与重试由共享滚动函数处理。
import { useEffect } from 'react'
import { useLocation } from 'react-router'
import { scrollToHeadlineHash } from '@/shared/lib/scrollToHeadline'

type UseScrollToHeadlineOptions = {
  content?: string
  isLoading?: boolean
}

// 加载中、正文为空或没有片段时不启动定位；依赖变化和卸载都会调用返回的取消函数。
export function useScrollToHeadline({
  content,
  isLoading,
}: UseScrollToHeadlineOptions) {
  const { hash } = useLocation()
  useEffect(() => {
    if (isLoading || !content || !hash) return
    const cancel = scrollToHeadlineHash(hash)
    return cancel
  }, [content, isLoading, hash])
}
