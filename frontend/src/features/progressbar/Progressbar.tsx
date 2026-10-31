// 将共享加载标记转换为延迟出现的进度条，进度使用定时模拟，不表示实际任务完成比例。
import { Progress } from '@/shared/components/ui/progress'
import { useEffect, useRef, useState } from 'react'
import { useProgressbarStore } from '@/features/progressbar/progressbarStore'

type TimeoutId = ReturnType<typeof setTimeout>
type IntervalId = ReturnType<typeof setInterval>

export default function Progressbar() {
  const loading = useProgressbarStore((state) => state.loading)

  const [value, setValue] = useState<number>(0)
  const [showLoadingbar, setShowLoadingbar] = useState<boolean>(false)

  const showTimerRef = useRef<TimeoutId | null>(null)
  const hideTimerRef = useRef<TimeoutId | null>(null)
  const intervalRef = useRef<IntervalId | null>(null)

  // 三个引用分别持有显示延迟、隐藏延迟和递增定时器，清理后置空避免继续持有句柄。
  const clearShowTimer = () => {
    if (showTimerRef.current) {
      clearTimeout(showTimerRef.current)
      showTimerRef.current = null
    }
  }

  const clearHideTimer = () => {
    if (hideTimerRef.current) {
      clearTimeout(hideTimerRef.current)
      hideTimerRef.current = null
    }
  }

  const clearIntervalTimer = () => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
  }

  // 此效果只在加载标记变化时运行，读取该次渲染的显示状态来决定结束动画。
  useEffect(() => {
    if (loading) {
      // 开始加载时取消待隐藏动作，避免旧隐藏回调影响当前显示周期。
      clearHideTimer()

      // 重置进度并延迟 100 毫秒显示，短于此时间的加载无需展示进度条。
      clearShowTimer()
      setValue(0)

      showTimerRef.current = setTimeout(() => {
        setValue(20) // 显示时以 20 作为模拟进度起点。
        setShowLoadingbar(true)
      }, 100)
    } else {
      // 加载结束先取消尚未执行的显示回调。
      clearShowTimer()

      if (showLoadingbar) {
        // 已显示时将进度设为 100，并等待 150 毫秒后隐藏和归零。
        setValue(100)

        clearHideTimer()
        hideTimerRef.current = setTimeout(() => {
          setShowLoadingbar(false)
          setValue(0)
        }, 150)
      } else {
        // 未显示时直接清空延迟与数值，不安排完成动画。
        clearHideTimer()
        setShowLoadingbar(false)
        setValue(0)
      }
    }

    return () => {
      // 加载标记变化或组件卸载时同时清理三个定时器。
      clearShowTimer()
      clearHideTimer()
      clearIntervalTimer()
    }
    // 依赖仅包含 loading，显示状态由本效果安排的回调修改。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading])

  // 可见期间每 200 毫秒随机增加不足 10 的进度，累计结果最多到 90。
  useEffect(() => {
    clearIntervalTimer()

    if (showLoadingbar) {
      intervalRef.current = setInterval(() => {
        setValue((prev) => {
          if (prev < 90) {
            // 达到或超过 90 后保持原值，等待加载结束分支写入完成值。
            const next = prev + Math.random() * 10
            return next > 90 ? 90 : next
          }
          return prev
        })
      }, 200)
    }

    return () => {
      clearIntervalTimer()
    }
  }, [showLoadingbar])

  if (!showLoadingbar) return null

  return (
    <Progress
      value={value}
      className="[&>div]:bg-brand-dark absolute top-0 left-0 z-50 h-1 rounded-none [&>div]:h-1 [&>div]:rounded-none"
    />
  )
}
