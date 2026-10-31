// 本文件用配置接口替身验证加载重试、实例版本过滤和语言应用。
// 虚拟计时器推进重试等待，预期告警同时核对原始内容，不访问真实配置接口。
import { expectDiagnostics } from '@/shared/testutil/expectedDiagnostics'
import * as configApi from '@/shared/lib/api/config'
import i18next from '@/shared/lib/i18n'
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
  type Mock,
} from 'vitest'
import { useConfigStore } from '@/shared/state/config'

vi.mock('@/shared/lib/api/config', () => ({
  getConfig: vi.fn(),
}))

// 共同响应省略偏好对象并令默认语言为空，使非语言用例只检查加载状态。
const baseConfig = {
  publicAccess: false,
  publicAccessEnvManaged: false,
  hideLinkMetadataSection: false,
  maxAssetUploadSizeBytes: 1000,
  enableRevision: false,
  enableLinkRefactor: false,
  enableApiKeyManagement: false,
  snapshotEnabled: false,
  smtpEnabled: false,
  httpRemoteUserEnabled: true,
  loginUrl: '',
  logoutUrl: '',
  userManagementUrl: '',
  defaultLanguage: '',
}

// 重置请求替身、启用虚拟时间并清理本组关注的启动状态。
beforeEach(() => {
  vi.resetAllMocks()
  vi.useFakeTimers()
  useConfigStore.setState({
    hasLoaded: false,
    configLoadSucceeded: false,
    httpRemoteUserEnabled: false,
    error: null,
  })
})

// 每例退出后恢复真实计时器，避免把后续文件的等待留在虚拟时间内。
afterEach(() => {
  vi.useRealTimers()
})

describe('loadConfig retry', () => {
  // 实例标识变化时允许版本从 100 降到 1；同一实例的版本 0 不能覆盖已接受的版本 1。
  it('accepts reset versions after server replacement and rejects older commits within one instance', async () => {
    useConfigStore.setState({ settingsInstance: 'old', settingsVersion: 100 })
    ;(configApi.getConfig as Mock).mockResolvedValueOnce({
      ...baseConfig,
      settingsInstance: 'new',
      settingsVersion: 1,
      publicAccess: true,
    })
    await useConfigStore.getState().loadConfig()
    expect(useConfigStore.getState().publicAccess).toBe(true)
    ;(configApi.getConfig as Mock).mockResolvedValueOnce({
      ...baseConfig,
      settingsInstance: 'new',
      settingsVersion: 0,
      publicAccess: false,
    })
    await useConfigStore.getState().loadConfig()
    expect(useConfigStore.getState().publicAccess).toBe(true)
    useConfigStore.setState({ settingsInstance: '', settingsVersion: 0 })
  })
  // 第一次请求拒绝，推进等待后第二次成功，状态应结束加载并保存代理认证模式。
  it('retries and succeeds after a transient /api/config failure', async () => {
    expectDiagnostics('warn', [
      [
        'Error loading configuration (attempt 1/3, retrying):',
        new Error('network blip'),
      ],
    ])
    ;(configApi.getConfig as Mock)
      .mockRejectedValueOnce(new Error('network blip'))
      .mockResolvedValueOnce(baseConfig)

    const loadPromise = useConfigStore.getState().loadConfig()
    await vi.runAllTimersAsync()
    await loadPromise

    expect(useConfigStore.getState().configLoadSucceeded).toBe(true)
    expect(useConfigStore.getState().hasLoaded).toBe(true)
    expect(useConfigStore.getState().httpRemoteUserEnabled).toBe(true)
    expect(configApi.getConfig).toHaveBeenCalledTimes(2)
  })

  // 所有请求均失败，逐条核对三次诊断，并确认已结束启动等待但配置未成功。
  it('gives up with configLoadSucceeded false after repeated failures', async () => {
    expectDiagnostics('warn', [
      [
        'Error loading configuration (attempt 1/3, retrying):',
        new Error('down'),
      ],
      [
        'Error loading configuration (attempt 2/3, retrying):',
        new Error('down'),
      ],
      [
        'Error loading configuration (attempt 3/3, giving up):',
        new Error('down'),
      ],
    ])
    ;(configApi.getConfig as Mock).mockRejectedValue(new Error('down'))

    const loadPromise = useConfigStore.getState().loadConfig()
    await vi.runAllTimersAsync()
    await loadPromise

    expect(useConfigStore.getState().configLoadSucceeded).toBe(false)
    expect(useConfigStore.getState().hasLoaded).toBe(true)
    expect(useConfigStore.getState().httpRemoteUserEnabled).toBe(false)
    expect((configApi.getConfig as Mock).mock.calls.length).toBeGreaterThan(1)
  })

  // 首次成功后直接完成，不再登记第二次配置读取。
  it('does not retry when the first attempt succeeds', async () => {
    ;(configApi.getConfig as Mock).mockResolvedValueOnce(baseConfig)

    await useConfigStore.getState().loadConfig()

    expect(useConfigStore.getState().configLoadSucceeded).toBe(true)
    expect(configApi.getConfig).toHaveBeenCalledTimes(1)
  })
})

describe('loadConfig defaultLanguage', () => {
  afterEach(async () => {
    // 语言实例是共享对象，每例结束后恢复英语，下一例可直接核对是否发生语言切换。
    await i18next.changeLanguage('en')
  })

  // 打包语言 de 同时写入存储并成为翻译实例当前语言。
  it('switches the UI language when defaultLanguage is a shipped language', async () => {
    ;(configApi.getConfig as Mock).mockResolvedValueOnce({
      ...baseConfig,
      defaultLanguage: 'de',
    })

    await useConfigStore.getState().loadConfig()

    expect(i18next.language).toBe('de')
    expect(useConfigStore.getState().defaultLanguage).toBe('de')
  })

  // 空语言不触发切换，翻译实例继续保持英语。
  it('leaves the UI language unchanged when defaultLanguage is empty', async () => {
    ;(configApi.getConfig as Mock).mockResolvedValueOnce(baseConfig)

    await useConfigStore.getState().loadConfig()

    expect(i18next.language).toBe('en')
  })

  // 未打包代码 xx 保存在配置状态中，但不应用到翻译实例，并产生一条明确告警。
  it('ignores an unrecognized defaultLanguage', async () => {
    expectDiagnostics('warn', [
      [
        'Configured default language "xx" is not shipped with the frontend, ignoring',
      ],
    ])
    ;(configApi.getConfig as Mock).mockResolvedValueOnce({
      ...baseConfig,
      defaultLanguage: 'xx',
    })

    await useConfigStore.getState().loadConfig()

    expect(i18next.language).toBe('en')
    expect(useConfigStore.getState().defaultLanguage).toBe('xx')
  })
})
