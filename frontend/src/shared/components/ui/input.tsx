// 原生输入与组合输入外框统一边框、焦点和禁用样式，不维护独立输入状态。
import * as React from 'react'

import { cn } from '@/shared/lib/utils'

// 时间输入保留浏览器原生选择器，通过 WebKit 伪元素定位图标并预留右侧空间。
// 这些选择器只影响支持对应伪元素的浏览器，其余原生输入属性仍原样传递。
const timeInputClasses =
  'relative min-w-0 pr-9 [&::-webkit-datetime-edit]:p-0 [&::-webkit-calendar-picker-indicator]:absolute [&::-webkit-calendar-picker-indicator]:right-3 [&::-webkit-calendar-picker-indicator]:h-4 [&::-webkit-calendar-picker-indicator]:w-4 [&::-webkit-calendar-picker-indicator]:cursor-pointer [&::-webkit-calendar-picker-indicator]:p-0 [&::-webkit-calendar-picker-indicator]:opacity-50 dark:[&::-webkit-calendar-picker-indicator]:invert disabled:[&::-webkit-calendar-picker-indicator]:cursor-not-allowed'

// 只有 type 严格为 time 时合并专用类名，调用方类名放在末尾参加冲突处理。
// ref 指向实际 input，value、onChange 和校验属性均由调用方控制。
const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<'input'>>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          'border-input file:text-interface-text placeholder:text-muted-foreground focus-visible:ring-ring flex h-9 w-full max-w-full min-w-0 rounded-md border bg-transparent px-3 py-1 text-base shadow-xs transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium focus-visible:ring-1 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
          type === 'time' && timeInputClasses,
          className,
        )}
        ref={ref}
        {...props}
      />
    )
  },
)
Input.displayName = 'Input'

// 组合外框按 flex 为输入与装饰节点分配空间，直接子输入去掉自身边框和焦点环。
// 外框在内部元素获得焦点时显示统一焦点环，文字区域以 flex-1 使用剩余宽度。
function InputGroup({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn(
        'border-input focus-within:ring-ring flex w-full max-w-full min-w-0 items-center rounded-md border shadow-xs focus-within:ring-1 [&>input]:w-0 [&>input]:flex-1 [&>input]:border-0 [&>input]:shadow-none [&>input]:focus-visible:ring-0',
        className,
      )}
      {...props}
    />
  )
}

export { Input, InputGroup }
