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

// Headers use the same content box as badges and bordered action buttons.
export function SettingsTableHeader({
  children,
  content = 'text',
}: {
  children: React.ReactNode
  content?: 'text' | 'role' | 'actions'
}) {
  return (
    <th scope="col" className="settings__table-header-cell">
      <span
        className={`settings__table-heading settings__table-heading--${content}`}
      >
        {children}
      </span>
    </th>
  )
}

// Column count controls total width; shared CSS owns the dimensions of each cell.
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

// Only overflowing text enters the tab order and opens a portaled full-text hint.
export function TableCellContent({ children }: { children: React.ReactNode }) {
  const contentRef = useRef<HTMLDivElement>(null)
  const [overflowText, setOverflowText] = useState('')

  useEffect(() => {
    const element = contentRef.current
    if (!element) return
    const measure = () => {
      setOverflowText(
        element.scrollWidth > element.clientWidth
          ? (element.textContent ?? '')
          : '',
      )
    }
    const resizeObserver =
      typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(measure)
    const mutationObserver = new MutationObserver(measure)
    measure()
    resizeObserver?.observe(element)
    window.addEventListener('resize', measure)
    mutationObserver.observe(element, {
      childList: true,
      subtree: true,
      characterData: true,
    })
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

// Semantic cells share overflow measurement and keyboard-accessible full text.
export function SettingsTableCell({
  children,
  className,
  ...props
}: ComponentProps<'td'>) {
  return (
    <td {...props} className={cn('settings__table-cell', className)}>
      <TableCellContent>{children}</TableCellContent>
    </td>
  )
}
