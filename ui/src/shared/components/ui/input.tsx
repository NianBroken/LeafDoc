import * as React from 'react'

import { cn } from '@/shared/lib/utils'

// Native time fields retain their picker and align its icon with other controls.
const timeInputClasses =
  'relative min-w-0 pr-9 [&::-webkit-datetime-edit]:p-0 [&::-webkit-calendar-picker-indicator]:absolute [&::-webkit-calendar-picker-indicator]:right-3 [&::-webkit-calendar-picker-indicator]:h-4 [&::-webkit-calendar-picker-indicator]:w-4 [&::-webkit-calendar-picker-indicator]:cursor-pointer [&::-webkit-calendar-picker-indicator]:p-0 [&::-webkit-calendar-picker-indicator]:opacity-50 dark:[&::-webkit-calendar-picker-indicator]:invert disabled:[&::-webkit-calendar-picker-indicator]:cursor-not-allowed'

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

// Adornments occupy their own flex slots so editable text never runs underneath them.
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
