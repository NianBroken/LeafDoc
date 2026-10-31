// 将日期格式函数接入 React 的语言和偏好订阅，相关值变化时重新生成格式函数集合。
// 相对时间仍读取日期库全局语言，本 Hook 不设置随时间流逝触发的重渲染计时器。

import { useUserSettingsStore } from '@/shared/state/userSettings'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  formatDateOnly,
  formatDateTime,
  formatRelativeTime,
  formatTimeOnly,
} from '@/shared/lib/formatDate'

// 组件获得的统一调用形状，每个函数接收可选时间字符串并返回显示文本。
export type DateTimeFormatters = {
  formatDateTime: (value?: string) => string
  formatDateOnly: (value?: string) => string
  formatTimeOnly: (value?: string) => string
  formatRelativeTime: (value?: string) => string
}

// 同时订阅界面语言、日期和时间模式，避免组件直接调用状态读取函数后无法更新。
export function useDateTimeFormat(): DateTimeFormatters {
  const { i18n } = useTranslation()
  const lng = i18n.language
  const dateFormat = useUserSettingsStore((s) => s.dateFormat)
  const timeFormat = useUserSettingsStore((s) => s.timeFormat)

  // 绝对格式函数捕获这一轮偏好，依赖不变时复用对象，减少下游引用变化。
  return useMemo(() => {
    const prefs = { dateFormat, timeFormat }
    return {
      formatDateTime: (value?: string) => formatDateTime(value, lng, prefs),
      formatDateOnly: (value?: string) => formatDateOnly(value, lng, prefs),
      formatTimeOnly: (value?: string) => formatTimeOnly(value, lng, prefs),
      formatRelativeTime,
    }
  }, [lng, dateFormat, timeFormat])
}
