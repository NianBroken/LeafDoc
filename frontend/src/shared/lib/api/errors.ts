// 本文件把接口结构错误、字段错误和普通异常转换为可展示的中文或当前语言提示。
// 翻译依据 errors 命名空间，未找到词条时保留服务端文本或调用方回退文本。
import i18next from '@/shared/lib/i18n'

// 服务端本地化错误包含稳定类别、原提示、翻译模板和可选字符串参数。
export type ApiLocalizedErrorDetail = {
  code: string
  message: string
  template: string
  args?: string[]
}

// 结构错误在响应顶层的 error 属性中保存详细对象。
export type ApiLocalizedErrorResponse = {
  error: ApiLocalizedErrorDetail
}

// 界面使用的统一错误形状，字段错误可与全局提示同时存在。
export type ApiUiError = {
  message: string
  code?: string
  fields?: Record<string, string>
}

// 包装服务端详细错误，保留原消息和模板供显示时按当前语言翻译。
// 参数数组直接保存传入引用，缺失时创建空数组，不执行深拷贝或运行时类型校验。
export class ApiLocalizedError extends Error {
  code: string
  template: string
  args: string[]

  constructor(detail: ApiLocalizedErrorDetail) {
    super(detail.message)
    this.name = 'ApiLocalizedError'
    this.code = detail.code
    this.template = detail.template
    this.args = detail.args ?? []
  }
}

// 仅检查两层对象和三个字符串字段，不校验 args 是否数组、成员类型或错误码是否已知。
export function isApiLocalizedErrorResponse(
  value: unknown,
): value is ApiLocalizedErrorResponse {
  if (!value || typeof value !== 'object') return false

  const error = (value as { error?: unknown }).error
  if (!error || typeof error !== 'object') return false

  const detail = error as Partial<ApiLocalizedErrorDetail>
  return (
    typeof detail.code === 'string' &&
    typeof detail.message === 'string' &&
    typeof detail.template === 'string'
  )
}

// 只接受当前类的实例，不把形状相似的普通对象自动转换为错误实例。
export function asApiLocalizedError(err: unknown): ApiLocalizedError | null {
  if (err instanceof ApiLocalizedError) {
    return err
  }
  return null
}

// 本地化错误优先按 page_not_found 错误码判断，其余对象只要 status 精确为数字 404 即接受。
// 本地化错误带其他错误码时直接返回 false，不继续检查它的 status 字段。
export function isPageNotFoundError(err: unknown): boolean {
  if (err instanceof ApiLocalizedError) {
    return err.code === 'page_not_found'
  }

  if (
    err &&
    typeof err === 'object' &&
    'status' in err &&
    (err as { status?: unknown }).status === 404
  ) {
    return true
  }

  return false
}

// 按原模板中的 %s 顺序替换参数，缺少参数时保留占位符，多余参数在末尾集中展示。
// 替换由回调完成，参数中的美元符号和 %s 不再当作替换规则或新占位符处理。
export function formatLocalizedErrorTemplate(
  template: string,
  args: string[] = [],
): string {
  if (!template) return ''

  let argIndex = 0
  const formatted = template.replace(/%s/g, () => {
    const nextArg = args[argIndex]
    argIndex += 1
    return nextArg ?? '%s'
  })

  if (argIndex >= args.length) {
    return formatted
  }

  return `${formatted} (${args.slice(argIndex).join(', ')})`
}

// 优先解释字段错误，再解释本地化错误实例，其他情况走通用消息提取。
// 字段提示以中文分号连接，得到空文本时才采用 fallback。
export function mapApiError(err: unknown, fallback: string): ApiUiError {
  const validation = readFieldErrors(err)
  if (validation)
    return {
      code: 'validation_error',
      fields: validation,
      message: Object.values(validation).join('；') || fallback,
    }
  const localized = asApiLocalizedError(err)
  if (localized) {
    const translated = i18next.t(localized.template, {
      ns: 'errors',
      defaultValue: localized.message || fallback,
    })
    const message = formatLocalizedErrorTemplate(translated, localized.args)

    return {
      message: message || fallback,
      code: localized.code,
    }
  }

  return {
    message: getErrorMessage(err, fallback),
  }
}

// readFieldErrors 只接受 validation_error 与数组 fields 的组合，其他形状返回 null。
// 合法条目按小写消息键翻译，相同字段后面的提示覆盖前面的提示，空数组返回空对象。
export function readFieldErrors(err: unknown): Record<string, string> | null {
  if (
    !err ||
    typeof err !== 'object' ||
    !('error' in err) ||
    err.error !== 'validation_error' ||
    !('fields' in err) ||
    !Array.isArray(err.fields)
  )
    return null
  const fields: Record<string, string> = {}
  for (const item of err.fields) {
    if (
      item &&
      typeof item.field === 'string' &&
      typeof item.message === 'string'
    )
      fields[item.field] = i18next.t(item.message.toLowerCase(), {
        ns: 'errors',
        defaultValue: item.message,
      })
  }
  return fields
}

// getErrorMessage 依次提取字段提示、Error 消息和回退文本，不读取任意对象的 message。
// 网络 TypeError 和 TimeoutError 使用固定翻译键，其他 Error 消息按小写键尝试翻译。
export function getErrorMessage(err: unknown, fallback: string): string {
  const fields = readFieldErrors(err)
  if (fields) return Object.values(fields).join('；') || fallback
  if (err instanceof Error && err.message) {
    if (
      err instanceof TypeError &&
      /fetch|network|load failed/i.test(err.message)
    )
      return i18next.t('network request failed', { ns: 'errors' })
    if (err.name === 'TimeoutError')
      return i18next.t('request timed out', { ns: 'errors' })
    return i18next.t(err.message.toLowerCase(), {
      ns: 'errors',
      defaultValue: err.message,
    })
  }
  return fallback
}
