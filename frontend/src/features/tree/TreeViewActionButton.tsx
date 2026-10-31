// 统一树工具栏按钮的提示、可访问名称和测试标识，并阻止点击继续冒泡到树行。
import { TooltipWrapper } from '@/shared/components/TooltipWrapper'
import { ComponentPropsWithoutRef, forwardRef } from 'react'

type TreeViewActionButtonProps = {
  actionName: string
  icon: React.ReactNode
  tooltip: string
} & ComponentPropsWithoutRef<'button'>

export const TreeViewActionButton = forwardRef<
  HTMLButtonElement,
  TreeViewActionButtonProps
>(function TreeViewActionButton(
  { onClick, icon, actionName, tooltip, type = 'button', ...props },
  ref,
) {
  return (
    <div className="group mr-2 flex">
      <TooltipWrapper label={tooltip} side="top" align="start">
        <button
          {...props}
          ref={ref}
          type={type}
          onClick={(e) => {
            // 先执行调用方动作，再阻止事件到达父级；不阻止按钮本身的默认行为。
            onClick?.(e)
            e.stopPropagation()
          }}
          className="btn-treeview"
          aria-label={tooltip}
          data-testid={`tree-view-action-button-${actionName}`}
        >
          {icon}
        </button>
      </TooltipWrapper>
    </div>
  )
})
