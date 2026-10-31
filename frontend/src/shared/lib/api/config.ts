// 本文件直接获取启动所需的公开配置，避免配置加载依赖尚未确定的认证刷新行为。
import type { GlobalPreferences } from '@/shared/state/userSettings'
import type { ApiLocalizedErrorResponse } from '@/shared/lib/api/errors'
import i18next from '@/shared/lib/i18n'
import { API_BASE_URL } from '@/shared/lib/config'
import {
  ApiLocalizedError,
  isApiLocalizedErrorResponse,
  mapApiError,
} from '@/shared/lib/api/errors'

// 非成功配置响应支持文本错误、本地化详细错误和普通 message 字段。
type ConfigErrorResponse = {
  error?: string | ApiLocalizedErrorResponse['error']
  message?: string
}

// 公开配置同时包含实例偏好、功能开关与输入限制，类型声明不执行运行时校验。
export type Config = {
  // 版本与实例标识供配置同步判断使用，缺失状态由消费方处理。
  settingsVersion?: number
  settingsInstance?: string
  preferences: GlobalPreferences
  publicAccess: boolean
  // 表示公开访问是否由启动配置固定，供设置界面显示只读来源。
  publicAccessEnvManaged: boolean
  // 兼容的目录显示布尔值与完整 tocMode 同时保留，具体优先级由配置消费者处理。
  alwaysShowToc: boolean
  tocMode: string
  editorLimit: number
  hideLinkMetadataSection: boolean
  maxAssetUploadSizeBytes: number
  maxAvatarUploadSizeBytes: number
  avatarAllowedExts: string[]
  enableRevision: boolean
  enableLinkRefactor: boolean
  enableApiKeyManagement: boolean
  snapshotEnabled: boolean
  smtpEnabled: boolean
  httpRemoteUserEnabled: boolean
  loginUrl: string
  logoutUrl: string
  userManagementUrl: string
  defaultLanguage: string
}

// 直接 fetch 配置，不调用 fetchWithAuth，不主动刷新会话或覆盖 credentials 默认规则。
// 失败先尝试本地化详细错误，再使用 error 字符串、message 和状态回退提示。
// 成功响应直接解析 JSON，网络异常与成功响应解析异常向上传播。
export async function getConfig(): Promise<Config> {
  const res = await fetch(`${API_BASE_URL}/api/config`)
  if (!res.ok) {
    const errorText = await res.text()
    const fallbackMessage = i18next.t('configLoad.fetchErrorFallback', {
      ns: 'common',
      status: res.status,
      statusText: res.statusText,
    })
    let errorBody: ConfigErrorResponse | null = null

    try {
      errorBody = errorText
        ? (JSON.parse(errorText) as ConfigErrorResponse)
        : null
    } catch {
      throw new Error(fallbackMessage)
    }

    if (isApiLocalizedErrorResponse(errorBody)) {
      throw new Error(
        mapApiError(new ApiLocalizedError(errorBody.error), fallbackMessage)
          .message,
      )
    }

    if (typeof errorBody?.error === 'string') {
      throw new Error(errorBody.error)
    }

    if (errorBody?.message) {
      throw new Error(errorBody.message)
    }

    throw new Error(fallbackMessage)
  }
  return await res.json()
}
