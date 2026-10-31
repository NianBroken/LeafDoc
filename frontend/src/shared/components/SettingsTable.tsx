// 设置表格统一列布局和单元格样式，并仅为溢出文字提供完整内容提示。
import { useEffect, useRef, useState } from 'react'
import type { ComponentProps, CSSProperties } from 'react'
import { cn } from '@/shared/lib/utils'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  TooltipProvider,
} from '@/shared/components/ui/tooltip'

type SettingsTableProps = ComponentProps<'table'> & { columns: number }

// SettingsTableHeader 生成列标题及内部文字容器，列语义通过 scope=col 表达。
export function SettingsTableHeader({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <th scope="col" className="settings__table-header-cell">
      <span className="settings__table-heading">{children}</span>
    </th>
  )
}

// SettingsTable 通过 CSS 变量和 col 元素数量表达列数，其余 table 属性原样透传。
// 自定义列数变量覆盖传入 style 中同名变量，TooltipProvider 使用三百毫秒延迟。
export function SettingsTable({
  columns,
  className,
  style,
  children,
  ...props
}: SettingsTableProps) {
  return (
    <TooltipProvider delayDuration={300}>
      <table
        {...props}
        className={cn('settings__table', className)}
        style={
          { ...style, '--settings-table-columns': columns } as CSSProperties
        }
      >
        <colgroup>
          {Array.from({ length: columns }, (_, index) => (
            <col key={index} />
          ))}
        </colgroup>
        {children}
      </table>
    </TooltipProvider>
  )
}

// TableCellContent 比较实际滚动宽度和可见宽度，只有溢出且有文字时显示提示并可聚焦。
// 提示使用 textContent，不复制子组件结构或渲染 HTML 字符串。
export function TableCellContent({ children }: { children: React.ReactNode }) {
  const contentRef = useRef<HTMLDivElement>(null)
  const [overflowText, setOverflowText] = useState('')

  useEffect(() => {
    const element = contentRef.current
    if (!element) return
    // 测量同时响应尺寸、窗口大小和子树文字变化，不在每次渲染主动重建观察器。
    const measure = () => {
      setOverflowText(
        element.scrollWidth > element.clientWidth
          ? (element.textContent ?? '')
          : '',
      )
    }
    const resizeObserver =
      typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(measure)
    // ResizeObserver 不可用时仍监听窗口变化，MutationObserver 在此直接要求环境提供。
    const mutationObserver = new MutationObserver(measure)
    measure()
    resizeObserver?.observe(element)
    window.addEventListener('resize', measure)
    mutationObserver.observe(element, {
      childList: true,
      subtree: true,
      characterData: true,
    })
    // 卸载时断开两个观察器并移除窗口监听，避免节点被持续引用。
    return () => {
      resizeObserver?.disconnect()
      window.removeEventListener('resize', measure)
      mutationObserver.disconnect()
    }
  }, [])

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div
          ref={contentRef}
          className="settings__table-text table-cell-text"
          tabIndex={overflowText ? 0 : undefined}
        >
          {children}
        </div>
      </TooltipTrigger>
      {overflowText && (
        <TooltipContent className="settings__table-tooltip">
          {overflowText}
        </TooltipContent>
      )}
    </Tooltip>
  )
}

// SettingsTableCell 文本模式使用溢出提示，controls 模式只提供控件布局容器。
// 控件自身的焦点和提示行为由其组件处理，不套用文字溢出测量。
export function SettingsTableCell({
  children,
  className,
  content = 'text',
  ...props
}: ComponentProps<'td'> & { content?: 'text' | 'controls' }) {
  return (
    <td {...props} className={cn('settings__table-cell', className)}>
      {content === 'controls' ? (
        <div className="settings__table-controls">{children}</div>
      ) : (
        <TableCellContent>{children}</TableCellContent>
      )}
    </td>
  )
}
