// 本文件用内存响应替换 fetch，检查认证刷新、一次重试、错误解码及写请求头。
// 用例自行设定配置是否已确认和用户快照，不访问运行中的账号或服务端会话表。
import '@/shared/lib/i18n'
import { useConfigStore } from '@/shared/state/config'
import { useSessionStore } from '@/shared/state/session'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { confirmPasswordReset, login } from '@/shared/lib/api/auth'
import { ApiError, ensureRefresh, fetchWithAuth } from '@/shared/lib/api/client'
import { ApiLocalizedError } from '@/shared/lib/api/errors'

// 单个响应只声明状态与可选 JSON 正文，未提供正文时由工厂返回空对象。
type MockResponseSpec = {
  status: number
  body?: unknown
}

// 按请求地址后缀匹配响应，数组配置按顺序消费，剩最后一项时继续复用该项。
// 例如先返回 401 再返回 200，可检查刷新后重试；未匹配路径固定返回 404。
function createFetchMock(
  routes: Record<string, MockResponseSpec | MockResponseSpec[]>,
) {
  const queues = new Map<string, MockResponseSpec[]>(
    Object.entries(routes).map(([path, spec]) => [
      path,
      Array.isArray(spec) ? [...spec] : [spec],
    ]),
  )

  return vi.fn(async (input: RequestInfo | URL) => {
    const url = typeof input === 'string' ? input : input.toString()
    const path = Object.keys(routes).find((p) => url.endsWith(p))
    if (!path) {
      return new Response(JSON.stringify({ error: 'unmapped_route' }), {
        status: 404,
      })
    }
    const queue = queues.get(path)!
    const spec = queue.length > 1 ? queue.shift()! : queue[0]
    return new Response(JSON.stringify(spec.body ?? {}), {
      status: spec.status,
      headers: { 'Content-Type': 'application/json' },
    })
  })
}

// 提取替身收到的地址，供断言某端点调用过、未调用或只调用一次。
function calledPaths(fetchMock: ReturnType<typeof createFetchMock>): string[] {
  return fetchMock.mock.calls.map(([input]) =>
    typeof input === 'string' ? input : String(input),
  )
}

// 内存中的管理员视图用于检查保留或清除状态，不对应数据库实际账号。
const testUser = {
  id: 'user-1',
  username: 'ivan',
  email: 'ivan@example.com',
  role: 'admin' as const,
}

// fetch 的全局替身在每例结束后解除，避免影响其他网络相关用例。
afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ensureRefresh', () => {
  beforeEach(() => {
    useConfigStore.setState({
      httpRemoteUserEnabled: false,
      configLoadSucceeded: true,
    })
    useSessionStore.setState({ user: null, accessTokenExpiresAt: null })
  })

  it('still attempts refresh when config is not confirmed (may be a wasted 422)', async () => {
    // 配置尚未确认时仍调用刷新端点；此例提供 422，要求拒绝结果与实际调用同时成立。
    useConfigStore.setState({ configLoadSucceeded: false })
    const fetchMock = createFetchMock({
      '/api/auth/refresh-token': { status: 422 },
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(ensureRefresh()).rejects.toThrow()

    expect(
      calledPaths(fetchMock).some((p) => p.endsWith('/api/auth/refresh-token')),
    ).toBe(true)
  })

  // 已确认使用代理身份时刷新直接完成，不向会话端点发送请求。
  it('does not call the refresh endpoint when httpRemoteUserEnabled is true', async () => {
    useConfigStore.setState({
      httpRemoteUserEnabled: true,
      configLoadSucceeded: true,
    })
    const fetchMock = createFetchMock({})
    vi.stubGlobal('fetch', fetchMock)

    await expect(ensureRefresh()).resolves.toBeUndefined()

    expect(fetchMock).not.toHaveBeenCalled()
  })

  // 已确认的 Cookie 会话模式接收完整刷新响应，确认请求确实到达刷新端点。
  it('calls the refresh endpoint for a confirmed session-auth deployment', async () => {
    const fetchMock = createFetchMock({
      '/api/auth/refresh-token': {
        status: 200,
        body: {
          accessTokenExpiresAt: Math.floor(Date.now() / 1000) + 3600,
          message: 'ok',
          user: testUser,
        },
      },
    })
    vi.stubGlobal('fetch', fetchMock)

    await ensureRefresh()

    expect(
      calledPaths(fetchMock).some((p) => p.endsWith('/api/auth/refresh-token')),
    ).toBe(true)
  })
})

describe('fetchWithAuth', () => {
  beforeEach(() => {
    useConfigStore.setState({
      httpRemoteUserEnabled: false,
      configLoadSucceeded: true,
      hasLoaded: true,
    })
    useSessionStore.setState({ user: null, accessTokenExpiresAt: null })
  })

  it('still succeeds when a preemptive refresh fails and config has not loaded', async () => {
    // 配置未确认、用户快照存在且到期时间为空，触发请求前的尝试刷新。
    // 刷新返回 422 后仍发送业务请求，业务成功时保留用户且不调用退出端点。
    // 本例不单独读取 CSRF Cookie，断言范围是请求地址和用户状态。
    useConfigStore.setState({ configLoadSucceeded: false })
    useSessionStore.setState({ user: testUser, accessTokenExpiresAt: null })

    const fetchMock = createFetchMock({
      '/api/some/path': { status: 200, body: { ok: true } },
      '/api/auth/refresh-token': { status: 422 },
      '/api/auth/logout': { status: 200 },
    })
    vi.stubGlobal('fetch', fetchMock)

    const result = await fetchWithAuth('/api/some/path')

    expect(result).toEqual({ ok: true })
    const paths = calledPaths(fetchMock)
    expect(paths.some((p) => p.endsWith('/api/auth/refresh-token'))).toBe(true)
    expect(paths.some((p) => p.endsWith('/api/auth/logout'))).toBe(false)
    expect(useSessionStore.getState().user).toEqual(testUser)
  })

  it('surfaces a genuine 401 as an error without forcing logout when config has not loaded', async () => {
    useConfigStore.setState({ configLoadSucceeded: false })
    useSessionStore.setState({
      user: testUser,
      // 到期时间放在一小时后，跳过请求前刷新，使失败来源限定为业务响应后的 401 分支。
      accessTokenExpiresAt: Math.floor(Date.now() / 1000) + 3600,
    })

    const fetchMock = createFetchMock({
      '/api/some/path': { status: 401 },
      '/api/auth/refresh-token': { status: 422 },
      '/api/auth/logout': { status: 200 },
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchWithAuth('/api/some/path')).rejects.toThrow()

    expect(
      calledPaths(fetchMock).some((p) => p.endsWith('/api/auth/logout')),
    ).toBe(false)
    expect(useSessionStore.getState().user).toEqual(testUser)
  })

  // 业务首次返回 401，刷新成功后同一路径返回 200，并要求只执行一次刷新。
  it('retries and succeeds after a 401 for a confirmed session-auth deployment', async () => {
    useConfigStore.setState({ configLoadSucceeded: true })
    useSessionStore.setState({
      user: testUser,
      accessTokenExpiresAt: Math.floor(Date.now() / 1000) + 3600,
    })

    const fetchMock = createFetchMock({
      '/api/some/path': [{ status: 401 }, { status: 200, body: { ok: true } }],
      '/api/auth/refresh-token': {
        status: 200,
        body: {
          accessTokenExpiresAt: Math.floor(Date.now() / 1000) + 3600,
          message: 'ok',
          user: testUser,
        },
      },
    })
    vi.stubGlobal('fetch', fetchMock)

    const result = await fetchWithAuth('/api/some/path')

    expect(result).toEqual({ ok: true })
    expect(
      calledPaths(fetchMock).filter((p) =>
        p.endsWith('/api/auth/refresh-token'),
      ),
    ).toHaveLength(1)
  })

  // 已确认会话模式下，业务 401 与刷新 422 组合触发退出端点和本地用户清除。
  it('still forces a logout when a confirmed session token is genuinely expired', async () => {
    useConfigStore.setState({ configLoadSucceeded: true })
    useSessionStore.setState({
      user: testUser,
      accessTokenExpiresAt: Math.floor(Date.now() / 1000) + 3600,
    })

    const fetchMock = createFetchMock({
      '/api/some/path': { status: 401 },
      '/api/auth/refresh-token': { status: 422 },
      '/api/auth/logout': { status: 200 },
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchWithAuth('/api/some/path')).rejects.toThrow()

    expect(
      calledPaths(fetchMock).some((p) => p.endsWith('/api/auth/logout')),
    ).toBe(true)
    expect(useSessionStore.getState().user).toBeNull()
  })
})

describe('confirmPasswordReset', () => {
  // 重置密码失败返回字段错误对象，调用方必须继续读取 fields，而非只拿到错误标识文本。
  it('re-throws a field-validation body untouched instead of the literal "validation_error"', async () => {
    const body = {
      error: 'validation_error',
      fields: [
        {
          field: 'newPassword',
          message: 'New password must be at least 8 characters long',
        },
      ],
    }
    const fetchMock = createFetchMock({
      '/api/auth/password/reset': { status: 400, body },
    })
    vi.stubGlobal('fetch', fetchMock)

    const err = await confirmPasswordReset('tok', 'short').then(
      () => {
        throw new Error('expected confirmPasswordReset to reject')
      },
      (e: unknown) => e,
    )

    // 保留对象的完整结构，不把它包装成 Error，便于表单错误处理器定位 newPassword 字段。
    expect(err).not.toBeInstanceOf(Error)
    expect(err).toEqual(body)
  })
})

// 对照统一认证客户端与登录请求的错误接口，分别核对对象类型、消息和 HTTP 状态。
describe('API error decoding', () => {
  const localized = {
    error: {
      code: 'page_not_found',
      message: 'Page not found',
      template: 'Page not found',
    },
  }
  const validation = {
    error: 'validation_error',
    fields: [{ field: 'title', message: 'Title is required' }],
  }

  // 直接控制原始响应文本，204 不附带正文，以符合 Response 构造的状态约束。
  function stubRawResponse(status: number, text: string) {
    const fetchMock = vi.fn(
      async () => new Response(status === 204 ? null : text, { status }),
    )
    vi.stubGlobal('fetch', fetchMock)
    return fetchMock
  }

  // 将拒绝值作为结果返回；意外成功直接使测试失败，防止无错误也通过后续检查。
  async function rejection(promise: Promise<unknown>): Promise<unknown> {
    return promise.then(
      () => {
        throw new Error('expected request to reject')
      },
      (e: unknown) => e,
    )
  }

  beforeEach(() => {
    useConfigStore.setState({
      httpRemoteUserEnabled: false,
      configLoadSucceeded: true,
      hasLoaded: true,
    })
    useSessionStore.setState({ user: null, accessTokenExpiresAt: null })
  })

  describe('fetchWithAuth', () => {
    // 字段错误保持服务器对象形状，调用方可继续按字段展示。
    it('re-throws a validation body untouched', async () => {
      stubRawResponse(400, JSON.stringify(validation))
      expect(await rejection(fetchWithAuth('/api/x'))).toEqual(validation)
    })

    // 结构化本地化错误保留稳定错误码，便于界面选择翻译。
    it('throws ApiLocalizedError for a localized error body', async () => {
      stubRawResponse(404, JSON.stringify(localized))
      const err = await rejection(fetchWithAuth('/api/x'))
      expect(err).toBeInstanceOf(ApiLocalizedError)
      expect((err as ApiLocalizedError).code).toBe('page_not_found')
    })

    // 普通字符串错误使用 ApiError，并保留服务端的 409 状态。
    it('throws ApiError with the status for a string error field', async () => {
      stubRawResponse(409, JSON.stringify({ error: 'conflict' }))
      const err = await rejection(fetchWithAuth('/api/x'))
      expect(err).toBeInstanceOf(ApiError)
      expect((err as ApiError).message).toBe('conflict')
      expect((err as ApiError).status).toBe(409)
    })

    // error 是空串时仍作为合法字符串处理，不替换为通用提示。
    it('keeps an empty string error field as the message', async () => {
      stubRawResponse(400, JSON.stringify({ error: '' }))
      const err = await rejection(fetchWithAuth('/api/x'))
      expect(err).toBeInstanceOf(ApiError)
      expect((err as ApiError).message).toBe('')
    })

    // 缺少 error 时读取字符串 message，同时保留原始响应状态。
    it('falls back to a string message field', async () => {
      stubRawResponse(500, JSON.stringify({ message: 'boom' }))
      const err = await rejection(fetchWithAuth('/api/x'))
      expect(err).toBeInstanceOf(ApiError)
      expect((err as ApiError).message).toBe('boom')
      expect((err as ApiError).status).toBe(500)
    })

    // 网关纯文本不能按 JSON 解码，结果使用通用服务提示并携带 502。
    it('uses the HTTP status for a non-JSON error body', async () => {
      stubRawResponse(502, 'Bad Gateway')
      const err = await rejection(fetchWithAuth('/api/x'))
      expect(err).toBeInstanceOf(ApiError)
      expect((err as ApiError).message).toBe('Service request failed')
      expect((err as ApiError).status).toBe(502)
    })

    // 空正文和无已知错误字段的 JSON 都回退到同一非空提示。
    it('uses the generic message for an empty or unrecognized error body', async () => {
      stubRawResponse(500, '')
      const empty = await rejection(fetchWithAuth('/api/x'))
      stubRawResponse(500, JSON.stringify({ unexpected: true }))
      const unknownShape = await rejection(fetchWithAuth('/api/x'))
      for (const err of [empty, unknownShape]) {
        expect(err).toBeInstanceOf(ApiError)
        expect((err as ApiError).message).not.toBe('')
        expect((err as ApiError).status).toBe(500)
      }
      expect((empty as ApiError).message).toBe(
        (unknownShape as ApiError).message,
      )
    })

    // 成功的空正文返回 null，删除调用不必强行得到 JSON 对象。
    it('resolves null for a successful response without a JSON body', async () => {
      stubRawResponse(204, '')
      expect(await fetchWithAuth('/api/x', { method: 'DELETE' })).toBeNull()
    })

    // 普通写入同时携带 Cookie、JSON 类型和从文档 Cookie 读取的 CSRF 值，正文原样传递。
    it('sends JSON content type, credentials and the CSRF header on writes', async () => {
      document.cookie = 'leafdoc_csrf=csrf-token-value'
      const fetchMock = stubRawResponse(200, '{}')
      await fetchWithAuth('/api/x', { method: 'POST', body: '{"a":1}' })
      const [url, init] = fetchMock.mock.calls[0] as unknown as [
        string,
        RequestInit,
      ]
      const headers = new Headers(init.headers)
      expect(url.endsWith('/api/x')).toBe(true)
      expect(init.credentials).toBe('include')
      expect(headers.get('Content-Type')).toBe('application/json')
      expect(headers.get('X-CSRF-Token')).toBe('csrf-token-value')
      expect(init.body).toBe('{"a":1}')
      document.cookie = 'leafdoc_csrf=; expires=Thu, 01 Jan 1970 00:00:00 GMT'
    })

    // 测试环境是 HTTP，两个名称并存时应读取普通 Cookie，不使用 HTTPS 名称的值。
    it('uses the HTTP cookie when a stale HTTPS cookie also exists on localhost', async () => {
      const cookie = vi
        .spyOn(document, 'cookie', 'get')
        .mockReturnValue(
          '__Host-leafdoc_csrf=https-token; leafdoc_csrf=http-token',
        )
      const fetchMock = stubRawResponse(200, '{}')
      await fetchWithAuth('/api/x', { method: 'PUT', body: '{}' })
      const [, init] = fetchMock.mock.calls[0] as unknown as [
        string,
        RequestInit,
      ]
      expect(new Headers(init.headers).get('X-CSRF-Token')).toBe('http-token')
      cookie.mockRestore()
    })

    // FormData 需要浏览器生成含边界的类型头，客户端不能预设 application/json。
    it('leaves the content type to the browser for FormData bodies', async () => {
      const fetchMock = stubRawResponse(200, '{}')
      const form = new FormData()
      await fetchWithAuth('/api/x', { method: 'POST', body: form })
      const [, init] = fetchMock.mock.calls[0] as unknown as [
        string,
        RequestInit,
      ]
      expect(new Headers(init.headers).has('Content-Type')).toBe(false)
      expect(init.body).toBe(form)
    })
  })

  describe('login requests', () => {
    // 登录入口同样保留字段错误对象，便于表单按字段显示失败原因。
    it('re-throws a validation body untouched', async () => {
      stubRawResponse(400, JSON.stringify(validation))
      expect(await rejection(login('a', 'b'))).toEqual(validation)
    })

    it('throws ApiLocalizedError for a localized error body', async () => {
      stubRawResponse(401, JSON.stringify(localized))
      expect(await rejection(login('a', 'b'))).toBeInstanceOf(ApiLocalizedError)
    })

    // 登录普通错误使用原生 Error，不附加统一认证客户端的 ApiError 类型。
    it('throws a plain Error (not ApiError) for a string error field', async () => {
      stubRawResponse(401, JSON.stringify({ error: 'invalid credentials' }))
      const err = await rejection(login('a', 'b'))
      expect(err).toBeInstanceOf(Error)
      expect(err).not.toBeInstanceOf(ApiError)
      expect((err as Error).message).toBe('invalid credentials')
    })

    // 非 JSON 和只有 message 的响应均使用登录回退提示，不采用该 message 字段。
    it('uses the login fallback for non-JSON and unrecognized bodies', async () => {
      stubRawResponse(500, 'oops')
      const nonJSON = await rejection(login('a', 'b'))
      stubRawResponse(500, JSON.stringify({ message: 'ignored' }))
      const unknownShape = await rejection(login('a', 'b'))
      for (const err of [nonJSON, unknownShape]) {
        expect(err).toBeInstanceOf(Error)
        expect(err).not.toBeInstanceOf(ApiError)
      }
      expect((nonJSON as Error).message).toBe((unknownShape as Error).message)
      expect((nonJSON as Error).message).not.toBe('ignored')
    })
  })
})

// 成功状态也区分空正文与损坏的 JSON，状态成功不代表正文能够正常解析。
describe('successful response integrity', () => {
  beforeEach(() => {
    useConfigStore.setState({ configLoadSucceeded: true })
    useSessionStore.setState({ user: null, accessTokenExpiresAt: null })
  })
  // 非空损坏正文向调用方抛出 SyntaxError，不能当作没有正文的成功结果。
  it('rejects malformed JSON instead of reporting an empty success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response('{broken', { status: 200 })),
    )
    await expect(fetchWithAuth('/api/x')).rejects.toBeInstanceOf(SyntaxError)
  })
  // 真正的 204 空正文仍返回 null，与上一个非空解析失败分支区分。
  it('accepts a genuinely empty successful response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(null, { status: 204 })),
    )
    await expect(fetchWithAuth('/api/x')).resolves.toBeNull()
  })
})
