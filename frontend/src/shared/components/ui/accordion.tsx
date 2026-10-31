// 折叠面板使用 Radix 的状态与事件入口，此文件统一边框、标题箭头和展开区域样式。
// 包装组件透传原语属性与 ref，不保存独立的展开状态。
import * as React from 'react'
import * as AccordionPrimitive from '@radix-ui/react-accordion'
import { ChevronDown } from 'lucide-react'

import { cn } from '@/shared/lib/utils'

// 根组件直接采用原语，单项或多项展开规则由调用方传入的根属性决定。
const Accordion = AccordionPrimitive.Root

// 每项下方绘制分隔边框，末项去掉该边框，调用方类名经统一合并规则处理。
const AccordionItem = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Item>
>(({ className, ...props }, ref) => (
  <AccordionPrimitive.Item
    ref={ref}
    className={cn('border-b last:border-b-0', className)}
    {...props}
  />
))
AccordionItem.displayName = 'AccordionItem'

// 标题放入 Header，触发按钮占满宽度，open 状态通过选择器旋转内部箭头。
// children 是实际标题内容，箭头在其后独立渲染并禁止收缩。
const AccordionTrigger = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Trigger>
>(({ className, children, ...props }, ref) => (
  <AccordionPrimitive.Header className="flex">
    <AccordionPrimitive.Trigger
      ref={ref}
      className={cn(
        'flex flex-1 items-center justify-between gap-3 py-2 text-left text-sm font-medium transition-all hover:underline [&[data-state=open]>svg]:rotate-180',
        className,
      )}
      {...props}
    >
      {children}
      <ChevronDown className="h-4 w-4 shrink-0 transition-transform duration-200" />
    </AccordionPrimitive.Trigger>
  </AccordionPrimitive.Header>
))
AccordionTrigger.displayName = AccordionPrimitive.Trigger.displayName

// 外层按开关状态切换高度动画并裁剪溢出，内层负责正文上下间距。
// className 同时参与外层和内层类名合并，调用方需考虑它会作用于两层元素。
const AccordionContent = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Content>
>(({ className, children, ...props }, ref) => (
  <AccordionPrimitive.Content
    ref={ref}
    className={cn(
      'data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down overflow-hidden text-sm',
      className,
    )}
    {...props}
  >
    <div className={cn('pt-1 pb-2', className)}>{children}</div>
  </AccordionPrimitive.Content>
))
AccordionContent.displayName = AccordionPrimitive.Content.displayName

export { Accordion, AccordionItem, AccordionTrigger, AccordionContent }
