// 保存实例运行配置，协调请求重试、并发响应顺序、偏好应用和认证运行模式读取。
import instanceDefaults from '@/shared/lib/instanceDefaults.json'
import { configureAuthRuntime } from '@/shared/lib/api/runtime'
import { useUserSettingsStore } from '@/shared/state/userSettings'
import { getConfig } from '@/shared/lib/api/config'
import {
  setAlwaysShowToc as setAlwaysShowTocApi,
  setPublicAccess as setPublicAccessApi,
} from '@/shared/lib/api/instanceSettings'
import {
  DEFAULT_AVATAR_ALLOWED_EXTS,
  DEFAULT_MAX_ASSET_UPLOAD_SIZE_BYTES,
  DEFAULT_MAX_AVATAR_UPLOAD_SIZE_BYTES,
} from '@/shared/lib/config'
import i18next, { getAvailableLanguages } from '@/shared/lib/i18n'
import { sleep } from '@/shared/lib/sleep'
import { create } from 'zustand'

type ConfigStore = {
  settingsInstance: string
  settingsVersion: number
  publicAccess: boolean
  publicAccessEnvManaged: boolean
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
  error: string | null
  loading: boolean
  hasLoaded: boolean
  // configLoadSucceeded 仅在配置请求成功时为 true，最终失败时设为 false。
  // hasLoaded 则在成功或最终失败后都为 true，用于结束启动加载界面。
  // 认证请求使用成功标志确认模式，避免在配置尚未确认时把刷新失败直接当作退出依据。
  configLoadSucceeded: boolean
  loadConfig: () => Promise<void>
  // 写入公开访问设置，成功后只同步本存储的 publicAccess，权限校验交给服务端。
  setPublicAccess: (enabled: boolean) => Promise<void>
  // 写入始终显示目录设置，成功后只更新 alwaysShowToc，不在此方法更新 tocMode。
  setAlwaysShowToc: (alwaysShow: boolean) => Promise<void>
}

// 每次 loadConfig 最多请求三次，两次重试等待分别为五百毫秒和一秒。
// 请求序号属于模块实例，每次调用使更早调用的后续结果失效，不主动中止网络请求。
const CONFIG_LOAD_MAX_ATTEMPTS = 3
const CONFIG_LOAD_RETRY_DELAYS_MS = [500, 1000]
let configRequestSequence = 0

export const useConfigStore = create<ConfigStore>((set, get) => ({
  settingsInstance: '',
  settingsVersion: 0,
  publicAccess: instanceDefaults.publicAccess,
  publicAccessEnvManaged: false,
  alwaysShowToc: false,
  tocMode: instanceDefaults.tocMode,
  editorLimit: 0,
  hideLinkMetadataSection: false,
  maxAssetUploadSizeBytes: DEFAULT_MAX_ASSET_UPLOAD_SIZE_BYTES,
  maxAvatarUploadSizeBytes: DEFAULT_MAX_AVATAR_UPLOAD_SIZE_BYTES,
  avatarAllowedExts: DEFAULT_AVATAR_ALLOWED_EXTS,
  enableRevision: false,
  enableLinkRefactor: false,
  enableApiKeyManagement: false,
  snapshotEnabled: true,
  smtpEnabled: false,
  httpRemoteUserEnabled: false,
  loginUrl: '',
  logoutUrl: '',
  userManagementUrl: '',
  defaultLanguage: instanceDefaults.language,
  error: null,
  loading: false,
  hasLoaded: false,
  configLoadSucceeded: false,

  loadConfig: async () => {
    // 已完成首次加载时后续刷新不显示初始加载状态，但仍清除旧错误提示。
    const sequence = ++configRequestSequence
    set({ loading: !get().hasLoaded, error: null })

    for (let attempt = 1; attempt <= CONFIG_LOAD_MAX_ATTEMPTS; attempt++) {
      try {
        const config = await getConfig()
        // 只接受最近一次调用。同一实例内低于当前版本的响应也被丢弃。
        if (sequence !== configRequestSequence) return
        if (
          config.settingsInstance === get().settingsInstance &&
          typeof config.settingsVersion === 'number' &&
          config.settingsVersion < get().settingsVersion
        )
          return
        // 上传大小只检查是否有限数值，不在这里强制为正；扩展名列表仅检查非空数组。
        const maxAssetUploadSizeBytes = Number.isFinite(
          config.maxAssetUploadSizeBytes,
        )
          ? config.maxAssetUploadSizeBytes
          : DEFAULT_MAX_ASSET_UPLOAD_SIZE_BYTES
        const maxAvatarUploadSizeBytes = Number.isFinite(
          config.maxAvatarUploadSizeBytes,
        )
          ? config.maxAvatarUploadSizeBytes
          : DEFAULT_MAX_AVATAR_UPLOAD_SIZE_BYTES
        const avatarAllowedExts =
          Array.isArray(config.avatarAllowedExts) &&
          config.avatarAllowedExts.length > 0
            ? config.avatarAllowedExts
            : DEFAULT_AVATAR_ALLOWED_EXTS

        // 先保存配置和成功状态，再应用偏好及语言；缺失可选字段采用各项回退值。
        set({
          settingsInstance: config.settingsInstance ?? get().settingsInstance,
          settingsVersion: config.settingsVersion ?? get().settingsVersion,
          publicAccess: config.publicAccess,
          publicAccessEnvManaged: config.publicAccessEnvManaged ?? false,
          alwaysShowToc: config.alwaysShowToc ?? false,
          tocMode: config.tocMode ?? (config.alwaysShowToc ? 'always' : 'auto'),
          editorLimit: config.editorLimit ?? 0,
          hideLinkMetadataSection: config.hideLinkMetadataSection,
          maxAssetUploadSizeBytes,
          maxAvatarUploadSizeBytes,
          avatarAllowedExts,
          enableRevision: config.enableRevision ?? false,
          enableLinkRefactor: config.enableLinkRefactor ?? false,
          enableApiKeyManagement: config.enableApiKeyManagement ?? false,
          snapshotEnabled: config.snapshotEnabled ?? false,
          smtpEnabled: config.smtpEnabled ?? false,
          httpRemoteUserEnabled: config.httpRemoteUserEnabled ?? false,
          loginUrl: config.loginUrl ?? '',
          logoutUrl: config.logoutUrl ?? '',
          userManagementUrl: config.userManagementUrl ?? '',
          defaultLanguage: config.defaultLanguage ?? instanceDefaults.language,
          error: null,
          hasLoaded: true,
          configLoadSucceeded: true,
          loading: false,
        })

        // 偏好 apply 本身也触发语言切换，这里的已知语言检查针对后续配置语言选择。
        if (config.preferences)
          useUserSettingsStore.getState().apply(config.preferences)
        const configuredLanguage =
          config.preferences?.language || config.defaultLanguage
        if (configuredLanguage) {
          const isKnownLanguage = getAvailableLanguages().some(
            (language) => language.code === configuredLanguage,
          )
          if (isKnownLanguage) {
            void i18next.changeLanguage(configuredLanguage)
          } else {
            console.warn(
              `Configured default language "${configuredLanguage}" is not shipped with the frontend, ignoring`,
            )
          }
        }
        return
      } catch (error) {
        // 旧调用失败不再写错误状态，当前调用仅在耗尽重试后结束加载并展示错误。
        if (sequence !== configRequestSequence) return
        if (attempt === CONFIG_LOAD_MAX_ATTEMPTS) {
          console.warn(
            `Error loading configuration (attempt ${attempt}/${CONFIG_LOAD_MAX_ATTEMPTS}, giving up):`,
            error,
          )
          set({
            error:
              error instanceof Error
                ? error.message
                : i18next.t('configLoad.errorFallback', { ns: 'common' }),
            hasLoaded: true,
            configLoadSucceeded: false,
            loading: false,
          })
          return
        }
        console.warn(
          `Error loading configuration (attempt ${attempt}/${CONFIG_LOAD_MAX_ATTEMPTS}, retrying):`,
          error,
        )
        await sleep(CONFIG_LOAD_RETRY_DELAYS_MS[attempt - 1])
      }
    }
  },

  // 状态以服务器返回值为准，请求失败直接拒绝，不预先翻转本地开关。
  setPublicAccess: async (enabled: boolean) => {
    const res = await setPublicAccessApi(enabled)
    set({ publicAccess: res.enabled })
  },

  setAlwaysShowToc: async (alwaysShow: boolean) => {
    const res = await setAlwaysShowTocApi(alwaysShow)
    set({ alwaysShowToc: res.alwaysShow })
  },
}))

// 向请求模块注册惰性读取器，每次取得当前配置，避免请求模块直接导入状态存储。
configureAuthRuntime(() => useConfigStore.getState())
