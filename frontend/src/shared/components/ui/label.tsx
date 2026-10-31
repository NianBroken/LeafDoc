// 原生 label 包装只负责文字样式，htmlFor、children 与其他标签属性由调用方提供。
import * as React from 'react'

import { cn } from '@/shared/lib/utils'

// 不添加额外业务属性，保留完整原生标签接口。
export interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {}

// ref 指向真实标签，peer-disabled 选择器配合相邻控件状态改变指针和透明度。
// 本组件不生成控件 ID，也不自动建立标签与输入之间的关联。
const Label = React.forwardRef<HTMLLabelElement, LabelProps>(
  ({ className, ...props }, ref) => (
    <label
      ref={ref}
      className={cn(
        'text-sm leading-none font-medium peer-disabled:cursor-not-allowed peer-disabled:opacity-70',
        className,
      )}
      {...props}
    />
  ),
)
Label.displayName = 'Label'

export { Label }
