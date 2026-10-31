// 本文件读取部署基础路径，集中提供上传显示默认值、媒体扩展名和字节格式化。
import i18next from 'i18next'

// 从页面 meta 标签读取基础路径，空值或未展开的双花括号模板视为根部署。
// 只补前导斜线并移除末尾斜线，不验证路径片段或 URL 编码。
function readBasePathFromMeta(): string {
  const raw =
    document
      .querySelector('meta[name="base-path"]')
      ?.getAttribute('content')
      ?.trim() ?? ''

  if (!raw || raw.includes('{{')) {
    return ''
  }

  const withSlash = raw.startsWith('/') ? raw : `/${raw}`
  return withSlash.replace(/\/+$/, '')
}

// 模块加载时读取一次页面配置，后续修改 meta 标签不会自动更新此常量。
export const BASE_PATH = readBasePathFromMeta()

// 接口地址与页面基础路径相同，不在此指定外部主机或额外 /api 前缀。
export const API_BASE_URL = (BASE_PATH ? `${BASE_PATH}` : '').replace(
  /\/+$/,
  '',
)

// 服务器配置尚未加载时使用的附件大小显示默认值，实际上传仍由服务端校验。
export const DEFAULT_MAX_ASSET_UPLOAD_SIZE_BYTES = 50 * 1024 * 1024

// 头像默认值对应后端头像模块的大小与扩展名，供服务端配置到达前使用。
// 它们不代替接口返回的实时配置，也不直接保证文件能够成功解码。
export const DEFAULT_MAX_AVATAR_UPLOAD_SIZE_BYTES = 5 * 1024 * 1024
export const DEFAULT_AVATAR_ALLOWED_EXTS = [
  '.gif',
  '.jpeg',
  '.jpg',
  '.png',
  '.webp',
]

// 按当前界面语言格式化数值，使用 1024 为进位基数，最大展示单位为 TiB。
// 字节级取整数，较大单位中小于十的值保留一位小数，其余不保留小数。
// 不校验负数、无穷值或 NaN，输入有效性由调用方保证。
export function formatBytes(bytes: number): string {
  const language = i18next.language || 'zh'
  const number = (value: number, digits: number) =>
    new Intl.NumberFormat(language, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    }).format(value)
  if (bytes < 1024) {
    return `${number(bytes, 0)} ${language.startsWith('zh') ? '字节' : 'B'}`
  }

  const units = ['KiB', 'MiB', 'GiB', 'TiB']
  let value = bytes
  let unitIndex = -1

  // 到达最高单位后停止换算，极大数值仍以 TiB 表示。
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }

  const fractionDigits = value >= 10 ? 0 : 1
  return `${number(value, fractionDigits)} ${units[unitIndex]}`
}

// 媒体类型识别共用的无点小写扩展名列表，不是上传许可或文件内容检测结果。
export const IMAGE_EXTENSIONS = [
  'png',
  'jpg',
  'jpeg',
  'gif',
  'webp',
  'bmp',
  'svg',
]

export const AUDIO_EXTENSIONS = ['mp3', 'wav', 'ogg', 'm4a', 'aac', 'flac']

export const VIDEO_EXTENSIONS = ['mp4', 'webm', 'ogv', 'mov', 'm4v']

export const PDF_EXTENSIONS = ['pdf']
