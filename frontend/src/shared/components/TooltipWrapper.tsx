// 根据移动端判断决定是否渲染工具提示，统一桌面提示的位置和外观。
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/shared/components/ui/tooltip'
import { useIsMobile } from '@/shared/lib/useIsMobile'
import clsx from 'clsx'
import { ReactNode } from 'react'

// asChild 为 true 时直接使用调用方子节点，不包裹额外 div。
type Props = {
  label: string
  children: ReactNode
  parentClassName?: string
  side?: 'left' | 'right' | 'top' | 'bottom'
  align?: 'start' | 'center' | 'end'
  asChild?: boolean
}

// TooltipWrapper 默认在顶部起始位置显示提示，移动端只返回内容容器或原子节点。
// 移动端分支不创建 Tooltip，因此 label 不会以浮层形式显示。
export function TooltipWrapper({
  label,
  children,
  side,
  align,
  parentClassName,
  asChild = false,
}: Props) {
  const tooltipSide = side || 'top'
  const tooltipAlign = align || 'start'
  const isMobile = useIsMobile()

  // 包裹模式应用 parentClassName，直接子节点模式不另行合并该类名。
  if (isMobile) {
    return asChild ? (
      <>{children}</>
    ) : (
      <div className={clsx('flex', parentClassName)}>{children}</div>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        {asChild ? (
          children
        ) : (
          <div className={clsx('flex', parentClassName)}>{children}</div>
        )}
      </TooltipTrigger>
      <TooltipContent
        side={tooltipSide}
        align={tooltipAlign}
        className={clsx(
          'tooltip-wrapper__content',
          'z-30 rounded-sm border px-2 py-1 text-xs shadow-sm',
          'bg-tooltip border-tooltip-border text-tooltip-text',
        )}
      >
        {label}
      </TooltipContent>
    </Tooltip>
  )
}
