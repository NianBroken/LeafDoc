import i18next from '@/shared/lib/i18n'

export type ApiLocalizedErrorDetail = {
  code: string
  message: string
  template: string
  args?: string[]
}

export type ApiLocalizedErrorResponse = {
  error: ApiLocalizedErrorDetail
}

export type ApiUiError = {
  message: string
  code?: string
  fields?: Record<string, string>
}

export class ApiLocalizedError extends Error {
  code: string
  template: string
  args: string[]

  constructor(detail: ApiLocalizedErrorDetail) {
    super(detail.message)
    this.name = 'ApiLocalizedError'
    this.code = detail.code
    this.template = detail.template
    this.args = detail.args ?? []
  }
}

export function isApiLocalizedErrorResponse(
  value: unknown,
): value is ApiLocalizedErrorResponse {
  if (!value || typeof value !== 'object') return false

  const error = (value as { error?: unknown }).error
  if (!error || typeof error !== 'object') return false

  const detail = error as Partial<ApiLocalizedErrorDetail>
  return (
    typeof detail.code === 'string' &&
    typeof detail.message === 'string' &&
    typeof detail.template === 'string'
  )
}

export function asApiLocalizedError(err: unknown): ApiLocalizedError | null {
  if (err instanceof ApiLocalizedError) {
    return err
  }
  return null
}

export function isPageNotFoundError(err: unknown): boolean {
  if (err instanceof ApiLocalizedError) {
    return err.code === 'page_not_found'
  }

  if (
    err &&
    typeof err === 'object' &&
    'status' in err &&
    (err as { status?: unknown }).status === 404
  ) {
    return true
  }

  return false
}

export function formatLocalizedErrorTemplate(
  template: string,
  args: string[] = [],
): string {
  if (!template) return ''

  let argIndex = 0
  const formatted = template.replace(/%s/g, () => {
    const nextArg = args[argIndex]
    argIndex += 1
    return nextArg ?? '%s'
  })

  if (argIndex >= args.length) {
    return formatted
  }

  return `${formatted} (${args.slice(argIndex).join(', ')})`
}

export function mapApiError(err: unknown, fallback: string): ApiUiError {
  const validation = readFieldErrors(err)
  if (validation)
    return {
      code: 'validation_error',
      fields: validation,
      message: Object.values(validation).join('；') || fallback,
    }
  const localized = asApiLocalizedError(err)
  if (localized) {
    const translated = i18next.t(localized.template, {
      ns: 'errors',
      defaultValue: localized.message || fallback,
    })
    const message = formatLocalizedErrorTemplate(translated, localized.args)

    return {
      message: message || fallback,
      code: localized.code,
    }
  }

  return {
    message: getErrorMessage(err, fallback),
  }
}

// Field messages remain attached to their inputs and also work in global notices.
export function readFieldErrors(err: unknown): Record<string, string> | null {
  if (
    !err ||
    typeof err !== 'object' ||
    !('error' in err) ||
    err.error !== 'validation_error' ||
    !('fields' in err) ||
    !Array.isArray(err.fields)
  )
    return null
  const fields: Record<string, string> = {}
  for (const item of err.fields) {
    if (
      item &&
      typeof item.field === 'string' &&
      typeof item.message === 'string'
    )
      fields[item.field] = i18next.t(item.message.toLowerCase(), {
        ns: 'errors',
        defaultValue: item.message,
      })
  }
  return fields
}

export function getErrorMessage(err: unknown, fallback: string): string {
  const fields = readFieldErrors(err)
  if (fields) return Object.values(fields).join('；') || fallback
  if (err instanceof Error && err.message) {
    if (
      err instanceof TypeError &&
      /fetch|network|load failed/i.test(err.message)
    )
      return i18next.t('network request failed', { ns: 'errors' })
    if (err.name === 'TimeoutError')
      return i18next.t('request timed out', { ns: 'errors' })
    return i18next.t(err.message.toLowerCase(), {
      ns: 'errors',
      defaultValue: err.message,
    })
  }
  return fallback
}
