// 工具提示包装保留 Radix 的提供器、根与触发器，集中定义提示正文弹层外观。
import * as React from 'react'
import * as TooltipPrimitive from '@radix-ui/react-tooltip'

import { cn } from '@/shared/lib/utils'

// 延迟和打开状态由消费方传给原语，此文件没有独立计时器或全局可见状态。
const TooltipProvider = TooltipPrimitive.Provider

const Tooltip = TooltipPrimitive.Root

const TooltipTrigger = TooltipPrimitive.Trigger

// 提示正文自动放入 Portal，默认离触发边八像素，宽高均受可用空间约束。
// 超长文字允许任意位置换行，超高正文纵向滚动，其余原语属性和 ref 原样传递。
const TooltipContent = React.forwardRef<
  React.ElementRef<typeof TooltipPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TooltipPrimitive.Content>
>(({ className, sideOffset = 8, ...props }, ref) => (
  <TooltipPrimitive.Portal>
    <TooltipPrimitive.Content
      ref={ref}
      sideOffset={sideOffset}
      className={cn(
        'bg-primary text-primary-foreground z-50 max-h-(--radix-tooltip-content-available-height) max-w-[min(24rem,var(--radix-tooltip-content-available-width))] origin-(--radix-tooltip-content-transform-origin) overflow-y-auto rounded-md px-3 py-1.5 text-xs wrap-anywhere',
        className,
      )}
      {...props}
    />
  </TooltipPrimitive.Portal>
))
TooltipContent.displayName = TooltipPrimitive.Content.displayName

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider }
