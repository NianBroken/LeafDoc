// Authentication workflows coordinate transport and pure session state.
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
const t = (key: string) => i18next.t(key, { ns: 'auth' })

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

function applyAuthResponse(data: AuthResponse) {
  const { setAccessTokenExpiresAt, setUser } = useSessionStore.getState()
  setAccessTokenExpiresAt(data.accessTokenExpiresAt)
  setUser(data.user)
}

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

// requestPasswordReset always resolves (never throws for an unknown
// identifier) — the backend deliberately returns the same response either
// way, so the UI must not try to distinguish "sent" from "no such user".
export async function requestPasswordReset(identifier: string): Promise<void> {
  await postLoginRequest<{ message: string }>('/api/auth/password/forgot', {
    identifier,
  })
}

export type PasswordResetConfirmResponse = {
  user: AuthResponse['user']
}

// confirmPasswordReset does NOT log the user in — a reset revokes every
// existing session for the account (see the backend's
// ConfirmPasswordResetUseCase), so the frontend sends the user to the login
// page afterward instead of calling applyAuthResponse.
export async function confirmPasswordReset(
  token: string,
  newPassword: string,
): Promise<PasswordResetConfirmResponse> {
  return postLoginRequest<PasswordResetConfirmResponse>(
    '/api/auth/password/reset',
    { token, newPassword },
  )
}

// acceptInvite sets the invited user's real password and, unlike
// confirmPasswordReset, logs them straight in — a freshly invited user has
// no prior session to worry about (see the backend's ConfirmInviteUseCase).
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

export async function logout() {
  try {
    await logoutRequest()
  } finally {
    useSessionStore.getState().clear()
  }
}

async function logoutRequest() {
  await endServerSession()
}
