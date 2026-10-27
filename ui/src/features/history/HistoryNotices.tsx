// Presentation elements shared by history loading, metadata and empty states.

import { type ApiUiError } from '@/shared/lib/api/errors'

import { type ReactNode } from 'react'

export function ErrorNotice({ error }: { error: ApiUiError }) {
  return (
    <div className="page-history__error-notice">
      <div className="page-history__error-title">{error.message}</div>
    </div>
  )
}

export function MetaChip({ children }: { children: ReactNode }) {
  return <span className="page-history__meta-chip">{children}</span>
}

export function ChangeChip({
  label,
  from,
  to,
}: {
  label: string
  from: string
  to: string
}) {
  return (
    <div className="page-history__change-chip">
      <span className="page-history__change-chip-label">{label}</span>
      <span className="page-history__change-chip-value">
        <span className="page-history__change-chip-from">{from}</span>
        <span className="page-history__change-chip-arrow" aria-hidden="true">
          →
        </span>
        <span className="page-history__change-chip-to">{to}</span>
      </span>
    </div>
  )
}

export function RevisionBadge({
  children,
  testId,
}: {
  children: ReactNode
  testId?: string
}) {
  return (
    <span className="history-sidebar__badge" data-testid={testId}>
      {children}
    </span>
  )
}

export function EmptyState({
  title,
  message,
}: {
  title: string
  message: string
}) {
  return (
    <div className="page-history__empty-state">
      <div className="page-history__empty-state-title">{title}</div>
      <div className="page-history__empty-state-message">{message}</div>
    </div>
  )
}
