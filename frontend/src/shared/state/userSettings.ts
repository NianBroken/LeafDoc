// 维护实例共享的语言与日期时间偏好，写入通过管理员设置接口串行发送。
import instanceDefaults from '@/shared/lib/instanceDefaults.json'
import { create } from 'zustand'
import { fetchWithAuth } from '@/shared/lib/api/client'
import i18next from '@/shared/lib/i18n'

// GlobalPreferences 是完整偏好响应，局部编辑使用 Partial 作为请求补丁。
export type GlobalPreferences = {
  language: string
  dateFormat: string
  timeFormat: string
}
// 初始值来自与后端保持一致的生成配置，服务器响应到达后覆盖这些显示值。
export const DEFAULT_PREFERENCES: GlobalPreferences = {
  language: instanceDefaults.language,
  dateFormat: instanceDefaults.dateFormat,
  timeFormat: instanceDefaults.timeFormat,
}
type Store = GlobalPreferences & {
  loaded: boolean
  apply: (value: GlobalPreferences) => void
  setLanguage: (value: string) => Promise<void>
  setDateFormat: (value: string) => Promise<void>
  setTimeFormat: (value: string) => Promise<void>
}
// pending 串联同一模块实例的写请求，前一请求失败不会阻止下一请求继续执行。
// 发起失败的调用方仍收到其自己的拒绝结果，队列只在开始下一任务时消化前次失败。
let pending: Promise<unknown> = Promise.resolve()
// save 等待队列后发送补丁，并用服务器返回的完整偏好覆盖本地状态。
// 不做乐观写入，失败时保留当前状态且把错误向上传递。
function save(patch: Partial<GlobalPreferences>): Promise<void> {
  const task = pending
    .catch(() => undefined)
    .then(async () => {
      const response = await fetchWithAuth<GlobalPreferences>(
        `/api/admin/settings/preferences`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(patch),
        },
      )
      useUserSettingsStore.getState().apply(response)
    })
  pending = task
  return task
}
export const useUserSettingsStore = create<Store>((set) => ({
  ...DEFAULT_PREFERENCES,
  loaded: false,
  // 完整替换三项偏好并标记已加载，语言切换异步发起，不等待其完成。
  apply: (value) => {
    set({
      language: value.language,
      dateFormat: value.dateFormat,
      timeFormat: value.timeFormat,
      loaded: true,
    })
    void i18next.changeLanguage(value.language)
  },
  setLanguage: (language) => save({ language }),
  setDateFormat: (dateFormat) => save({ dateFormat }),
  setTimeFormat: (timeFormat) => save({ timeFormat }),
}))
