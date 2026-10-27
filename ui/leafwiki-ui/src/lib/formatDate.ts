import { format as formatWithPattern, formatDistanceToNow } from 'date-fns'
import { useUserSettingsStore } from '@/stores/userSettings'
import { dateFnsLocale } from './dateLocale'

/**
 * Relative time such as "3 days ago" / "vor 3 Tagen". The locale follows the
 * active UI language via date-fns' global default options (see dateLocale.ts).
 */
export function formatRelativeTime(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (isNaN(date.getTime())) return ''
  return formatDistanceToNow(date, { addSuffix: true })
}

export type DateTimeFormatPrefs = {
  dateFormat: string
  timeFormat: string
}

// Explicit instance-wide presets use stable patterns independently from device settings.
const DATE_PATTERNS: Record<string, string> = {
  iso: 'yyyy-MM-dd',
  dmy_dot: 'dd.MM.yyyy',
  mdy_slash: 'MM/dd/yyyy',
  dmy_slash: 'dd/MM/yyyy',
}

const TIME_PATTERNS: Record<string, string> = {
  '24h': 'HH:mm',
  '12h': 'h:mm a',
}

function currentPrefs(): DateTimeFormatPrefs {
  const s = useUserSettingsStore.getState()
  return { dateFormat: s.dateFormat, timeFormat: s.timeFormat }
}

function toDate(value?: string): Date | null {
  if (!value) return null
  const date = new Date(value)
  return isNaN(date.getTime()) ? null : date
}

function datePart(
  date: Date,
  lng: string | undefined,
  dateFormat: string,
): string {
  const pattern = DATE_PATTERNS[dateFormat]
  if (pattern) return formatWithPattern(date, pattern)
  return formatWithPattern(date, DATE_PATTERNS.iso, {
    locale: dateFnsLocale(lng),
  })
}

function timePart(
  date: Date,
  lng: string | undefined,
  timeFormat: string,
): string {
  const pattern = TIME_PATTERNS[timeFormat]
  if (pattern) {
    // Day-period labels follow the instance language; 24-hour time uses numeric fields.
    const options = { locale: dateFnsLocale(lng) }
    return formatWithPattern(date, pattern, options)
  }
  return formatWithPattern(date, TIME_PATTERNS['24h'], {
    locale: dateFnsLocale(lng),
  })
}

/** Absolute date + time. Honours the user's date/time format preference. */
export function formatDateTime(
  value?: string,
  lng?: string,
  prefs: DateTimeFormatPrefs = currentPrefs(),
): string {
  const date = toDate(value)
  if (!date) return ''
  return `${datePart(date, lng, prefs.dateFormat)} ${timePart(date, lng, prefs.timeFormat)}`
}

/** Absolute date only. Honours the user's date format preference. */
export function formatDateOnly(
  value?: string,
  lng?: string,
  prefs: DateTimeFormatPrefs = currentPrefs(),
): string {
  const date = toDate(value)
  if (!date) return ''
  return datePart(date, lng, prefs.dateFormat)
}

/** Absolute time only. Honours the user's time format preference. */
export function formatTimeOnly(
  value?: string,
  lng?: string,
  prefs: DateTimeFormatPrefs = currentPrefs(),
): string {
  const date = toDate(value)
  if (!date) return ''
  return timePart(date, lng, prefs.timeFormat)
}
