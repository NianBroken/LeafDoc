// 本文件协调认证请求和前端会话状态，登录类请求直接使用 fetch，避免触发通用自动刷新。
import i18next from '@/shared/lib/i18n'
import { useSessionStore } from '@/shared/state/session'
import { API_BASE_URL } from '@/shared/lib/config'
import {
  ApiError,
  endServerSession,
  throwIfStructuredError,
  stringField,
  type AuthResponse,
} from '@/shared/lib/api/client'
// 认证提示在使用时读取当前语言。
const t = (key: string) => i18next.t(key, { ns: 'auth' })

// 请求当前用户并携带 Cookie，非成功状态抛出含状态码的 ApiError，不自动刷新。
// 成功 JSON 的 null 或 undefined 统一返回 null，其他结构不在此验证。
export async function fetchMe(): Promise<AuthResponse['user'] | null> {
  const res = await fetch(`${API_BASE_URL}/api/auth/me`, {
    credentials: 'include',
  })
  if (!res.ok) {
    throw new ApiError(`/api/auth/me returned ${res.status}`, res.status)
  }
  const user = await res.json()
  return user ?? null
}

// 用 JSON POST 发送登录、找回密码和邀请等公开认证请求。
// 失败优先抛结构错误，其次读取 error 字符串，非法 JSON 或无提示时使用登录回退文本。
async function postLoginRequest<T>(path: string, body: object): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (!res.ok) {
    let errorBody: unknown = null
    try {
      errorBody = await res.json()
    } catch {
      throw new Error(t('login.errorFallback'))
    }

    throwIfStructuredError(errorBody)

    const message = stringField(errorBody, 'error')
    if (message !== undefined) {
      throw new Error(message)
    }

    throw new Error(t('login.errorFallback'))
  }

  return (await res.json()) as T
}

// 先保存到期时间再保存用户，两个状态更新按此顺序调用，不在本函数校验响应。
function applyAuthResponse(data: AuthResponse) {
  const { setAccessTokenExpiresAt, setUser } = useSessionStore.getState()
  setAccessTokenExpiresAt(data.accessTokenExpiresAt)
  setUser(data.user)
}

// 将账号标识与密码原样交给登录接口，成功后应用会话状态并返回完整响应。
export async function login(
  identifier: string,
  password: string,
): Promise<AuthResponse> {
  const data = await postLoginRequest<AuthResponse>('/api/auth/login', {
    identifier,
    password,
  })

  applyAuthResponse(data)
  return data
}

// requestPasswordReset 提交找回请求并丢弃成功响应，不根据返回结果区分账号是否存在。
// 网络异常、非成功状态和响应解析失败仍会向上传播，不保证所有调用都成功完成。
export async function requestPasswordReset(identifier: string): Promise<void> {
  await postLoginRequest<{ message: string }>('/api/auth/password/forgot', {
    identifier,
  })
}

// 密码重置确认只约定返回用户，不包含可直接应用的访问令牌到期值。
export type PasswordResetConfirmResponse = {
  user: AuthResponse['user']
}

// confirmPasswordReset 提交令牌与新密码，返回结果但不调用会话状态更新。
// 后续跳转由调用方执行，本函数自身不跳转到登录页。
export async function confirmPasswordReset(
  token: string,
  newPassword: string,
): Promise<PasswordResetConfirmResponse> {
  return postLoginRequest<PasswordResetConfirmResponse>(
    '/api/auth/password/reset',
    { token, newPassword },
  )
}

// acceptInvite 提交邀请令牌与密码，成功后应用返回的认证信息并返回完整响应。
export async function acceptInvite(
  token: string,
  newPassword: string,
): Promise<AuthResponse> {
  const data = await postLoginRequest<AuthResponse>('/api/auth/invite/accept', {
    token,
    newPassword,
  })
  applyAuthResponse(data)
  return data
}

// 不论服务端退出成功与否都清空前端会话，服务端退出错误仍向调用方传播。
export async function logout() {
  try {
    await logoutRequest()
  } finally {
    useSessionStore.getState().clear()
  }
}

// 复用统一退出请求的 Cookie 和 CSRF 处理，不重复实现传输规则。
async function logoutRequest() {
  await endServerSession()
}
