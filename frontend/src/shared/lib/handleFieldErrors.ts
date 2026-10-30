// Field validation and global notifications use the same error interpretation.
import { mapApiError } from '@/shared/lib/api/errors'
import { toast } from 'sonner'

export function handleFieldErrors(
  err: unknown,
  setFieldErrors: ((errors: Record<string, string>) => void) | undefined,
  fallbackMessage: string,
) {
  const mapped = mapApiError(err, fallbackMessage)
  if (mapped.fields) setFieldErrors?.(mapped.fields)
  toast.error(mapped.message)
}
