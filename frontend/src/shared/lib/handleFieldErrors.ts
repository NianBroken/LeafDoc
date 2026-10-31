// 字段提示与全局错误通知共用 API 错误映射，避免对同一错误使用不同解释。
import { mapApiError } from '@/shared/lib/api/errors'
import { toast } from 'sonner'

// 将错误映射后按需交给字段回调，并始终显示一条全局错误通知。
// 没有字段错误时不清空先前字段状态，回调缺失时只显示全局通知。
export function handleFieldErrors(
  err: unknown,
  setFieldErrors: ((errors: Record<string, string>) => void) | undefined,
  fallbackMessage: string,
) {
  const mapped = mapApiError(err, fallbackMessage)
  if (mapped.fields) setFieldErrors?.(mapped.fields)
  toast.error(mapped.message)
}
