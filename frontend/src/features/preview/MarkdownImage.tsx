import { versionAssetSrc } from '@/shared/lib/assetSrc'
import { DIALOG_IMAGE_PREVIEW } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useMemo } from 'react'

type MarkdownImageProps = React.ImgHTMLAttributes<HTMLImageElement> & {
  resolveAssetUrl?: (src: string) => string
}

// Markdown 图片使用带版本的资源地址，并在普通点击时打开站内预览对话框。
function shouldOpenPreview(e: React.MouseEvent<HTMLImageElement>) {
  // 仅未按修饰键的鼠标左键用于站内预览。
  if (e.button !== 0) return false
  if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return false
  return true
}

function shouldOpenInNewTab(e: React.MouseEvent<HTMLImageElement>) {
  // Ctrl 或 Command 加鼠标左键采用新标签页打开图片资源。
  return e.button === 0 && (e.metaKey || e.ctrlKey)
}

export function MarkdownImage({
  src = '',
  style,
  alt,
  width,
  resolveAssetUrl,
  ...rest
}: MarkdownImageProps) {
  // 资源解析函数先处理相对附件路径，随后追加缓存版本参数。
  const openDialog = useDialogsStore((s) => s.openDialog)
  const resolvedSrc = useMemo(
    () => resolveAssetUrl?.(src) ?? src,
    [resolveAssetUrl, src],
  )
  const versionedSrc = useMemo(
    () => versionAssetSrc(resolvedSrc),
    [resolvedSrc],
  )

  return (
    <img
      // 空资源地址不设置 src，也不触发预览光标。
      src={versionedSrc || undefined}
      alt={alt}
      style={{
        // 将图片设为行内块，避免预设样式把同一 Markdown 段落中的相邻文字换到新行。
        display: 'inline-block',
        // 清除图片块默认上下边距，段落间距由外围段落样式单独提供。
        marginTop: 0,
        marginBottom: 0,
        ...style,
        cursor: versionedSrc ? 'zoom-in' : undefined,
        ...(width
          ? {
              width,
              height: 'auto',
            }
          : {}),
      }}
      draggable={false}
      {...rest}
      onClick={(e) => {
        rest.onClick?.(e)

        if (!versionedSrc) return

        // 图片位于链接内部时保留链接的导航行为，不拦截为预览对话框。
        if (e.currentTarget.closest('a')) {
          return
        }

        if (shouldOpenInNewTab(e)) {
          window.open(versionedSrc, '_blank', 'noopener,noreferrer')
          return
        }

        if (!shouldOpenPreview(e)) {
          return
        }

        e.preventDefault()
        openDialog(DIALOG_IMAGE_PREVIEW, { src: versionedSrc, alt })
      }}
    />
  )
}
