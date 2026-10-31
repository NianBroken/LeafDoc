// 本文件统一请求携带凭据、CSRF 请求头、会话刷新和接口错误解释。
// 请求成功后的 JSON 只作类型断言，不执行响应结构校验。
import type { SessionUser } from '@/shared/contracts/auth'
import i18next from '@/shared/lib/i18n'
import { useSessionStore } from '@/shared/state/session'
import { authRuntime } from './runtime'
import { API_BASE_URL } from '@/shared/lib/config'
import {
  ApiLocalizedError,
  isApiLocalizedErrorResponse,
} from '@/shared/lib/api/errors'

// 认证提示在调用时读取当前语言，不在模块加载时固定翻译结果。
const t = (key: string) => i18next.t(key, { ns: 'auth' })

// 登录和刷新响应携带访问令牌失效秒值与用户，不把令牌正文保存到此结构。
export type AuthResponse = {
  accessTokenExpiresAt: number
  message: string
  user: SessionUser
}

// 刷新最多等待十五秒，令牌剩余不超过六十秒时请求前尝试刷新。
const REFRESH_TIMEOUT_MS = 15000
const REFRESH_THRESHOLD_SECONDS = 60

// ApiError 保留非成功响应的状态码，供上层区分认证、缺失资源等错误。
export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

// 从当前协议对应的可读 Cookie 取 CSRF 值，解码失败保留原始文本。
// 无 document 或无匹配项返回 null，不回退读取另一协议的 Cookie。
export function getCsrfTokenFromCookie(): string | null {
  if (typeof document === 'undefined') return null

  // HTTPS 使用带 __Host- 前缀的名称，HTTP 使用普通名称，分别匹配完整 Cookie 边界。
  const hostMatch =
    globalThis.location?.protocol === 'https:'
      ? document.cookie.match(/(?:^|;\s*)__Host-leafdoc_csrf=([^;]+)/)
      : document.cookie.match(/(?:^|;\s*)leafdoc_csrf=([^;]+)/)

  if (!hostMatch) return null
  try {
    return decodeURIComponent(hostMatch[1])
  } catch {
    return hostMatch[1]
  }
}

// throwIfStructuredError 对 validation_error 对象原样抛出，不在这里校验 fields 内容。
// 可识别的本地化结构包装成 ApiLocalizedError，其他输入直接结束，由调用方继续解释。
export function throwIfStructuredError(errorBody: unknown): void {
  if (
    errorBody &&
    typeof errorBody === 'object' &&
    (errorBody as { error?: unknown }).error === 'validation_error'
  ) {
    throw errorBody
  }
  if (isApiLocalizedErrorResponse(errorBody)) {
    throw new ApiLocalizedError(errorBody.error)
  }
}

// stringField 只读取对象中指定字符串属性，空字符串也作为有效结果返回。
export function stringField(
  errorBody: unknown,
  field: 'error' | 'message',
): string | undefined {
  if (!errorBody || typeof errorBody !== 'object') return undefined
  const value = (errorBody as Record<string, unknown>)[field]
  return typeof value === 'string' ? value : undefined
}

// fetchWithAuth 按当前认证模式发送带 Cookie 的请求，成功时返回解析后的 JSON。
// 空白响应返回 null，T 只约束调用方的静态类型，不验证服务端数据形状。
// retry 仅控制收到 401 后是否刷新重试，不控制请求前的到期刷新判断。
export async function fetchWithAuth<T = unknown>(
  path: string,
  options: RequestInit = {},
  retry = true,
): Promise<T> {
  const store = useSessionStore.getState()
  const sessionLogout = store.clear
  const config = authRuntime.readConfig()
  const httpRemoteUserEnabled = config.httpRemoteUserEnabled
  const configLoadSucceeded = config.configLoadSucceeded

  // 复制传入请求头，非 FormData 一律设 JSON 类型，包括未带正文的读取请求。
  const headers = new Headers(options.headers || {})
  if (!(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  const method = (options.method || 'GET').toUpperCase()
  if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
    const csrfToken = getCsrfTokenFromCookie()
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken)
    }
  }

  // 非 FormData 对象正文直接 JSON.stringify，不按 Blob 等其他对象类型分流。
  let originalBody = options.body
  if (
    options.body &&
    typeof options.body === 'object' &&
    !(options.body instanceof FormData)
  ) {
    originalBody = JSON.stringify(options.body)
  }

  // 每次重试复用已构造的头和正文，credentials 在展开 options 后固定为 include。
  const doFetch = async (): Promise<Response> => {
    return fetch(`${API_BASE_URL}${path}`, {
      ...options,
      credentials: 'include',
      headers,
      body: originalBody,
    })
  }

  if (
    shouldRefreshBeforeRequest(
      store.user,
      store.accessTokenExpiresAt,
      httpRemoteUserEnabled,
    )
  ) {
    try {
      await ensureRefresh()
    } catch {
      // 仅在配置已成功加载时将刷新失败判为会话失效并清理登录状态。
      // 模式尚未确认时继续实际请求，让响应决定是否可访问，不先强制退出。
      if (configLoadSucceeded) {
        await clearSessionState(sessionLogout)
        throw new Error(t('apiErrors.unauthorized'))
      }
    }
  }

  // 用户主动中止原样传播，fetch 抛出的 TypeError 转成当前语言网络提示。
  const request = async () => {
    try {
      return await doFetch()
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') throw err
      if (err instanceof TypeError)
        throw new Error(i18next.t('network request failed', { ns: 'errors' }))
      throw err
    }
  }
  let res = await request()

  // 非远程用户模式收到 401 时最多执行这一轮刷新和重试，不递归调用自身。
  if (res.status === 401 && retry && !httpRemoteUserEnabled) {
    try {
      await ensureRefresh()
      res = await request()
    } catch {
      if (configLoadSucceeded) {
        await clearSessionState(sessionLogout)
        throw new Error(t('apiErrors.unauthorized'))
      }
      // 未确认模式下保留当前非成功响应，继续标准错误处理，不循环刷新或返回伪成功。
    }
  }

  // 非成功响应先解析结构，非法 JSON 使用状态码通用提示，不向界面暴露原始 HTML。
  if (!res.ok) {
    const errorText = await res.text()
    let errorBody: unknown = null
    try {
      errorBody = errorText ? JSON.parse(errorText) : null
    } catch {
      throw new ApiError(httpFailureMessage(res.status), res.status)
    }

    throwIfStructuredError(errorBody)

    const message =
      stringField(errorBody, 'error') ?? stringField(errorBody, 'message')
    if (message !== undefined) {
      throw new ApiError(message, res.status)
    }

    throw new ApiError(httpFailureMessage(res.status), res.status)
  }

  const text = await res.text()
  if (!text.trim()) return null as T
  return JSON.parse(text) as T
}

// 全局共享同一个进行中的刷新 Promise，使并发请求等待同一刷新结果。
declare global {
  var __leafdocRefreshPromise: Promise<void> | null
}

if (typeof globalThis.__leafdocRefreshPromise === 'undefined') {
  globalThis.__leafdocRefreshPromise = null
}

// 远程用户模式或没有用户不刷新，已登录但缺失到期时间时立即尝试刷新。
// 有到期秒值时与当前整秒比较，已过期和临近到期均进入刷新。
function shouldRefreshBeforeRequest(
  user: AuthResponse['user'] | null,
  accessTokenExpiresAt: number | null,
  httpRemoteUserEnabled: boolean,
): boolean {
  if (httpRemoteUserEnabled || user === null) {
    return false
  }

  if (accessTokenExpiresAt === null) {
    return true
  }

  const nowInSeconds = Math.floor(Date.now() / 1000)
  return accessTokenExpiresAt - nowInSeconds <= REFRESH_THRESHOLD_SECONDS
}

// 先请求服务端退出，不论请求成功与否都清理前端状态；退出错误仍会向上传播。
async function clearSessionState(sessionLogout: () => void) {
  try {
    await endServerSession()
  } finally {
    sessionLogout()
  }
}

// ensureRefresh 在普通会话模式下复用进行中的刷新，完成或失败后清空全局引用。
export function ensureRefresh(): Promise<void> {
  const { httpRemoteUserEnabled } = authRuntime.readConfig()
  // 当前配置启用远程用户时直接成功结束。此处不要求配置已加载成功，
  // 未确认模式仍允许尝试刷新，失败是否触发退出由 fetchWithAuth 的调用分支判断。
  if (httpRemoteUserEnabled) {
    return Promise.resolve()
  }

  if (!globalThis.__leafdocRefreshPromise) {
    // finally 派生出的 Promise 就是共享对象，所有等待者接收相同的完成或失败结果。
    globalThis.__leafdocRefreshPromise = refreshAccessToken().finally(() => {
      globalThis.__leafdocRefreshPromise = null
    })
  }
  return globalThis.__leafdocRefreshPromise
}

// refreshAccessToken 用独立控制器约束刷新请求，成功后依次更新到期秒值与用户。
// 此调用只带 Cookie，不通过 fetchWithAuth，也不附加通用写入路径的 CSRF 头。
async function refreshAccessToken() {
  const store = useSessionStore.getState()
  const controller = new AbortController()
  const timeoutId = window.setTimeout(
    () => controller.abort(),
    REFRESH_TIMEOUT_MS,
  )

  try {
    const res = await fetch(`${API_BASE_URL}/api/auth/refresh-token`, {
      method: 'POST',
      credentials: 'include',
      signal: controller.signal,
    })

    if (!res.ok) {
      throw new ApiError(t('apiErrors.refreshFailed'), res.status)
    }

    const data: AuthResponse = await res.json()
    // 响应类型在此只作静态声明，实际值未经结构检查就依次写入共享会话状态。
    store.setAccessTokenExpiresAt(data.accessTokenExpiresAt)
    store.setUser(data.user)
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new Error(t('apiErrors.refreshTimedOut'))
    }

    throw error
  } finally {
    // 网络成功、服务端失败和中止均清除计时器，避免请求结束后继续触发中止。
    window.clearTimeout(timeoutId)
  }
}

// endServerSession 直接发送带 Cookie 与可用 CSRF 头的退出请求，不触发自动刷新。
// 这里只验证 HTTP 成功状态，不读取返回正文，也不直接清理前端状态。
export async function endServerSession(): Promise<void> {
  const headers = new Headers()
  const csrfToken = getCsrfTokenFromCookie()
  if (csrfToken) headers.set('X-CSRF-Token', csrfToken)
  const response = await fetch(`${API_BASE_URL}/api/auth/logout`, {
    method: 'POST',
    credentials: 'include',
    headers,
  })
  if (!response.ok)
    throw new ApiError(t('apiErrors.requestFailed'), response.status)
}

// httpFailureMessage 按已知状态选择翻译键，五百段使用服务失败，其余使用请求失败。
// 不拼接状态正文，避免把无法解析的 HTML 页面作为界面消息显示。
function httpFailureMessage(status: number): string {
  const keys: Record<number, string> = {
    400: 'invalid request input',
    401: 'authentication required',
    403: 'permission denied',
    404: 'resource not found',
    409: 'resource conflict',
    413: 'uploaded file too large',
    422: 'invalid request input',
    429: 'too many requests',
  }
  return i18next.t(
    keys[status] ??
      (status >= 500 ? 'service request failed' : 'request failed'),
    { ns: 'errors' },
  )
}
