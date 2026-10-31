import { FormInput } from '@/shared/components/FormInput'
import { mapApiError } from '@/shared/lib/api/errors'
import { suggestSlug } from '@/shared/lib/api/pages'
import i18next from '@/shared/lib/i18n'
import { useDebounce } from '@/shared/lib/useDebounce'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

// 输入框的值由调用方维护，组件负责标题防抖、服务端建议和手动编辑状态。
type Props = {
  title: string
  slug: string
  parentId: string
  testid?: string
  currentId?: string
  /**
   * 初次处理时，防抖标题若与该值完全相同，就保留调用方传入的路径段。
   * 该分支只处理一次，后续标题变化仍能触发建议，避免打开编辑窗口即覆盖现有名称。
   */
  initialTitle?: string
  enableSlugSuggestion?: boolean
  onSlugChange: (slug: string) => void
  onSlugTouchedChange?: (touched: boolean) => void
  onSlugLoadingChange?: (loading: boolean) => void
  onLastSlugTitleChange?: (title: string) => void
  error?: string
  allowedHotkeys?: string
}

// SlugInputWithSuggestion 在用户尚未手动编辑路径段时，按防抖后的标题请求名称建议。
// 建议由服务端处理有效名称与同级冲突，组件不在本地推导可用路径段。
export function SlugInputWithSuggestion({
  title,
  slug,
  currentId,
  testid,
  parentId,
  initialTitle,
  enableSlugSuggestion = true,
  onSlugChange,
  onSlugTouchedChange,
  onSlugLoadingChange,
  onLastSlugTitleChange,
  error,
  allowedHotkeys,
}: Props) {
  // 手动改动后保持已编辑状态，本次组件挂载期间不再由标题自动覆盖路径段。
  const [slugTouched, setSlugTouched] = useState(false)
  const debouncedTitle = useDebounce(title, 300)
  // 记录初始同标题分支是否已处理，不随组件重渲染重置。
  const initialHandledRef = useRef(false)

  useEffect(() => {
    if (!enableSlugSuggestion || slugTouched || debouncedTitle.trim() === '') {
      return
    }

    // 首次标题与初值一致时只通知已处理标题，不请求建议或改动现有路径段。
    // 已存在的数字避让后缀因此能随编辑窗口保持原值。
    if (
      !initialHandledRef.current &&
      initialTitle !== undefined &&
      debouncedTitle === initialTitle
    ) {
      initialHandledRef.current = true
      onLastSlugTitleChange?.(debouncedTitle)
      return
    }
    initialHandledRef.current = true

    // 请求期间通知调用方加载状态，成功时即使建议为空也按返回值更新受控输入。
    // 这里没有请求序号或清理取消逻辑，完成的请求直接调用传入的状态回调。
    const generateSlug = async () => {
      try {
        onSlugLoadingChange?.(true)
        const suggestion = await suggestSlug(
          parentId,
          debouncedTitle,
          currentId,
        )
        onSlugChange(suggestion)
        onLastSlugTitleChange?.(debouncedTitle)
      } catch (err) {
        // 错误映射为当前界面语言的提示，不以本地猜测名称替代服务端结果。
        const mapped = mapApiError(
          err,
          i18next.t('slugInput.generateErrorFallback', { ns: 'editor' }),
        )
        toast.error(mapped.message)
      } finally {
        onSlugLoadingChange?.(false)
      }
    }

    generateSlug()
  }, [
    debouncedTitle,
    slugTouched,
    parentId,
    currentId,
    initialTitle,
    onSlugLoadingChange,
    onSlugChange,
    onLastSlugTitleChange,
    enableSlugSuggestion,
  ])

  // 用户编辑立即上交新值，并通知手动编辑状态，后续标题变化不再发起建议请求。
  const handleChange = (val: string) => {
    onSlugChange(val)
    setSlugTouched(true)
    onSlugTouchedChange?.(true)
  }

  return (
    <FormInput
      label={i18next.t('slugInput.label', { ns: 'editor' })}
      value={slug}
      onChange={handleChange}
      placeholder={i18next.t('slugInput.placeholder', { ns: 'editor' })}
      testid={testid}
      error={error}
      allowedHotkeys={allowedHotkeys}
    />
  )
}
