import type { SessionUser } from '@/shared/contracts/auth'
import i18next from '@/shared/lib/i18n'
import { useSessionStore } from '@/shared/state/session'
import { authRuntime } from './runtime'
import { API_BASE_URL } from '@/shared/lib/config'
import {
  ApiLocalizedError,
  isApiLocalizedErrorResponse,
} from '@/shared/lib/api/errors'

const t = (key: string) => i18next.t(key, { ns: 'auth' })

export type AuthResponse = {
  accessTokenExpiresAt: number
  message: string
  user: SessionUser
}

const REFRESH_TIMEOUT_MS = 15000
const REFRESH_THRESHOLD_SECONDS = 60

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export function getCsrfTokenFromCookie(): string | null {
  if (typeof document === 'undefined') return null

  // Local HTTP and HTTPS deployments can coexist on one host with distinct cookies.
  const hostMatch =
    globalThis.location?.protocol === 'https:'
      ? document.cookie.match(/(?:^|;\s*)__Host-leafwiki_csrf=([^;]+)/)
      : document.cookie.match(/(?:^|;\s*)leafwiki_csrf=([^;]+)/)

  if (!hostMatch) return null
  try {
    return decodeURIComponent(hostMatch[1])
  } catch {
    return hostMatch[1]
  }
}

// Field-validation errors ({ error: 'validation_error', fields: [...] }) are
// re-thrown untouched so callers can surface per-field messages instead of
// the literal string "validation_error"; localized errors become
// ApiLocalizedError. Both are surfaced the same way by every request path.
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

// stringField returns errorBody[field] when it is a string (including "").
export function stringField(
  errorBody: unknown,
  field: 'error' | 'message',
): string | undefined {
  if (!errorBody || typeof errorBody !== 'object') return undefined
  const value = (errorBody as Record<string, unknown>)[field]
  return typeof value === 'string' ? value : undefined
}

// fetchWithAuth sends an authenticated request and resolves the parsed JSON
// body (null when the response has none). T only types the result for the
// caller; the body is not validated at runtime.
export async function fetchWithAuth<T = unknown>(
  path: string,
  options: RequestInit = {},
  retry = true,
): Promise<T> {
  const store = useSessionStore.getState()
  const sessionLogout = store.clear
  const config = authRuntime.readConfig()
  const authDisabled = config.authDisabled
  const httpRemoteUserEnabled = config.httpRemoteUserEnabled
  const configLoadSucceeded = config.configLoadSucceeded

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

  let originalBody = options.body
  if (
    options.body &&
    typeof options.body === 'object' &&
    !(options.body instanceof FormData)
  ) {
    originalBody = JSON.stringify(options.body)
  }

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
      authDisabled,
      httpRemoteUserEnabled,
    )
  ) {
    try {
      await ensureRefresh()
    } catch {
      // A refresh failure only proves the session is really gone once the
      // auth mode is confirmed (configLoadSucceeded) — otherwise this could
      // just as easily be a header-auth deployment whose refresh-token call
      // was always going to 422 (accessTokenExpiresAt is never set in that
      // mode, so the check above fires on every request). Forcing this
      // request to fail on that guess — instead of letting the real request
      // decide, the same way it already does when httpRemoteUserEnabled is
      // *confirmed* true — is what caused GitHub #1407 (spurious
      // CSRF-token-missing logouts, autosave breaking mid-edit).
      if (configLoadSucceeded) {
        await clearSessionState(sessionLogout)
        throw new Error(t('apiErrors.unauthorized'))
      }
    }
  }

  let res = await doFetch()

  if (res.status === 401 && retry && !authDisabled && !httpRemoteUserEnabled) {
    try {
      await ensureRefresh()
      res = await doFetch()
    } catch {
      if (configLoadSucceeded) {
        await clearSessionState(sessionLogout)
        throw new Error(t('apiErrors.unauthorized'))
      }
      // Unconfirmed mode: don't force a logout, but don't fabricate success
      // either — `res` still holds the original 401 from above, so it falls
      // through to the generic !res.ok handling below and surfaces as a
      // normal error instead of a silent retry loop.
    }
  }

  if (!res.ok) {
    const errorText = await res.text()
    let errorBody: unknown = null
    try {
      errorBody = errorText ? JSON.parse(errorText) : null
    } catch {
      throw new ApiError(errorText || t('apiErrors.requestFailed'), res.status)
    }

    throwIfStructuredError(errorBody)

    const message =
      stringField(errorBody, 'error') ?? stringField(errorBody, 'message')
    if (message !== undefined) {
      throw new ApiError(message, res.status)
    }

    throw new ApiError(t('apiErrors.requestFailed'), res.status)
  }

  const text = await res.text()
  if (!text.trim()) return null as T
  return JSON.parse(text) as T
}

declare global {
  var __leafwikiRefreshPromise: Promise<void> | null
}

if (typeof globalThis.__leafwikiRefreshPromise === 'undefined') {
  globalThis.__leafwikiRefreshPromise = null
}

function shouldRefreshBeforeRequest(
  user: AuthResponse['user'] | null,
  accessTokenExpiresAt: number | null,
  authDisabled: boolean,
  httpRemoteUserEnabled: boolean,
): boolean {
  if (authDisabled || httpRemoteUserEnabled || user === null) {
    return false
  }

  if (accessTokenExpiresAt === null) {
    return true
  }

  const nowInSeconds = Math.floor(Date.now() / 1000)
  return accessTokenExpiresAt - nowInSeconds <= REFRESH_THRESHOLD_SECONDS
}

async function clearSessionState(sessionLogout: () => void) {
  try {
    await endServerSession()
  } finally {
    sessionLogout()
  }
}

export function ensureRefresh(): Promise<void> {
  const { authDisabled, httpRemoteUserEnabled } = authRuntime.readConfig()
  // Session-token refresh only applies to session/JWT auth — skip it
  // whenever that's confirmed false, instead of relying on every caller to
  // have already checked httpRemoteUserEnabled itself. Deliberately NOT
  // gated on configLoadSucceeded: attempting a refresh while the auth mode
  // is still unconfirmed is harmless by itself (worst case one wasted call),
  // and skipping it here entirely would permanently disable refresh for
  // session-auth deployments that hit one bad /api/config fetch. The actual
  // risk — treating an unconfirmed-mode refresh failure as confirmed
  // unauthorized and forcing a logout — is guarded at the reaction site in
  // fetchWithAuth instead, not here.
  if (authDisabled || httpRemoteUserEnabled) {
    return Promise.resolve()
  }

  if (!globalThis.__leafwikiRefreshPromise) {
    globalThis.__leafwikiRefreshPromise = refreshAccessToken().finally(() => {
      globalThis.__leafwikiRefreshPromise = null
    })
  }
  return globalThis.__leafwikiRefreshPromise
}

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
    store.setAccessTokenExpiresAt(data.accessTokenExpiresAt)
    store.setUser(data.user)
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new Error(t('apiErrors.refreshTimedOut'))
    }

    throw error
  } finally {
    window.clearTimeout(timeoutId)
  }
}

// Session termination uses the same cookie and CSRF policy as other writes.
export async function endServerSession(): Promise<void> {
  if (authRuntime.readConfig().authDisabled) return
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
