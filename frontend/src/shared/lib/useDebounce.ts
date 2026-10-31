// 将频繁变化的输入延迟发布为状态值，首次渲染直接使用输入初值。
import { useEffect, useState } from 'react'

// value 或 delay 改变时取消上一计时器，只有最新计时器到期才更新返回值。
// 卸载时同样取消等待任务，不对对象输入作深比较。
export function useDebounce<T>(value: T, delay: number) {
  const [debouncedValue, setDebouncedValue] = useState(value)

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)

    // 清理函数覆盖依赖变化和组件卸载，避免旧输入延迟覆盖新一轮值。
    return () => {
      clearTimeout(handler)
    }
  }, [value, delay])

  return debouncedValue
}
