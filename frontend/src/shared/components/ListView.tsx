// 提供列表外框、标题、项目和状态行，数据加载与选中状态由调用方维护。
import { type ElementType, type ReactNode } from 'react'

// as 控制外层标签或组件，未提供时使用 aside，其余区块以共享类名组织布局。
type ListViewProps = {
  header?: ReactNode
  footer?: ReactNode
  children: ReactNode
  as?: ElementType
  className?: string
  contentClassName?: string
  testId?: string
}

type ListViewItemProps = {
  children: ReactNode
  active?: boolean
  className?: string
  onClick?: () => void
  testId?: string
}

type ListViewStatusProps = {
  children: ReactNode
  error?: boolean
  className?: string
}

// ListView 总是保留内容容器，header 和 footer 只有值为真时才渲染对应区域。
export function ListView({
  header,
  footer,
  children,
  as: Component = 'aside',
  className = '',
  contentClassName = '',
  testId,
}: ListViewProps) {
  return (
    <Component className={`list-view ${className}`.trim()} data-testid={testId}>
      {header ? <div className="list-view__header">{header}</div> : null}
      <div className={`list-view__content ${contentClassName}`.trim()}>
        {children}
      </div>
      {footer ? <div className="list-view__footer">{footer}</div> : null}
    </Component>
  )
}

// ListViewHeader 只提供标题样式，不决定标题层级或添加交互。
export function ListViewHeader({ children }: { children: ReactNode }) {
  return <div className="list-view__title">{children}</div>
}

// ListViewList 为条目集合提供布局 div，不自动生成 ul 或 li 语义。
export function ListViewList({ children }: { children: ReactNode }) {
  return <div className="list-view__list">{children}</div>
}

// ListViewItem 使用 type=button 避免误提交外围表单，active 仅控制类名。
// 点击原样调用回调，不在内部切换选中状态。
export function ListViewItem({
  children,
  active = false,
  className = '',
  onClick,
  testId,
}: ListViewItemProps) {
  return (
    <button
      type="button"
      className={`list-view__item ${active ? 'list-view__item--active' : ''} ${className}`.trim()}
      onClick={onClick}
      data-testid={testId}
    >
      {children}
    </button>
  )
}

// ListViewStatus 按 error 标志切换状态文字样式，不自动记录日志或发起重试。
export function ListViewStatus({
  children,
  error = false,
  className = '',
}: ListViewStatusProps) {
  return (
    <div
      className={`list-view__status ${error ? 'list-view__status--error' : ''} ${className}`.trim()}
    >
      {children}
    </div>
  )
}
