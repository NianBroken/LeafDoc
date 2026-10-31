// 本文件将时间字符串转换为相对时间或按实例偏好显示的本地日期时间。
// 未指定时区，Date 解析与格式化沿用浏览器环境的日期和时区规则。
import { format as formatWithPattern, formatDistanceToNow } from 'date-fns'
import { useUserSettingsStore } from '@/shared/state/userSettings'
import { dateFnsLocale } from '@/shared/lib/dateLocale'

/**
 * 返回带过去或未来后缀的相对时间，使用日期库当前全局语言。
 * 空输入和无法解析的日期返回空串，不设置定时刷新，也不接受显式语言参数。
 */
export function formatRelativeTime(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (isNaN(date.getTime())) return ''
  return formatDistanceToNow(date, { addSuffix: true })
}

// 格式函数接受的显示偏好，不包含时区或语言，语言由独立参数提供。
export type DateTimeFormatPrefs = {
  dateFormat: string
  timeFormat: string
}

// 日期排列和分隔符由明确模式决定，不根据设备区域自动改变。
const DATE_PATTERNS: Record<string, string> = {
  iso: 'yyyy-MM-dd',
  dmy_dot: 'dd.MM.yyyy',
  mdy_slash: 'MM/dd/yyyy',
  dmy_slash: 'dd/MM/yyyy',
}

// 二十四小时模式只显示数字，十二小时模式包含由日期语言决定的时段文字。
const TIME_PATTERNS: Record<string, string> = {
  '24h': 'HH:mm',
  '12h': 'h:mm a',
}

// 非 Hook 函数直接读取当前状态，调用此函数本身不会订阅偏好变化。
function currentPrefs(): DateTimeFormatPrefs {
  const s = useUserSettingsStore.getState()
  return { dateFormat: s.dateFormat, timeFormat: s.timeFormat }
}

// 空值和无效日期统一返回 null，其他字符串按 JavaScript Date 的规则解析。
function toDate(value?: string): Date | null {
  if (!value) return null
  const date = new Date(value)
  return isNaN(date.getTime()) ? null : date
}

// 已知日期模式直接格式化，未知模式回退 ISO 排列，不将未知文本当作格式串执行。
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

// 已知时间模式使用指定日期语言，未知模式回退二十四小时制。
function timePart(
  date: Date,
  lng: string | undefined,
  timeFormat: string,
): string {
  const pattern = TIME_PATTERNS[timeFormat]
  if (pattern) {
    // 语言对象决定十二小时模式的时段文字，数字时分仍采用明确格式。
    const options = { locale: dateFnsLocale(lng) }
    return formatWithPattern(date, pattern, options)
  }
  return formatWithPattern(date, TIME_PATTERNS['24h'], {
    locale: dateFnsLocale(lng),
  })
}

/** 按传入或当前偏好拼接日期与时间，两部分之间使用一个空格，无效输入返回空串。 */
export function formatDateTime(
  value?: string,
  lng?: string,
  prefs: DateTimeFormatPrefs = currentPrefs(),
): string {
  const date = toDate(value)
  if (!date) return ''
  return `${datePart(date, lng, prefs.dateFormat)} ${timePart(date, lng, prefs.timeFormat)}`
}

/** 只返回日期部分，默认偏好在调用时读取，日期解析失败不抛出格式化错误。 */
export function formatDateOnly(
  value?: string,
  lng?: string,
  prefs: DateTimeFormatPrefs = currentPrefs(),
): string {
  const date = toDate(value)
  if (!date) return ''
  return datePart(date, lng, prefs.dateFormat)
}

/** 只返回时间部分，采用显式语言与时间偏好，不对输入额外作时区换算。 */
export function formatTimeOnly(
  value?: string,
  lng?: string,
  prefs: DateTimeFormatPrefs = currentPrefs(),
): string {
  const date = toDate(value)
  if (!date) return ''
  return timePart(date, lng, prefs.timeFormat)
}
