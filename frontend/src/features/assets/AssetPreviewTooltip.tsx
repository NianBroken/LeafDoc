// 此悬停提示按 URL 最后一个点后的文字识别图片，其他附件展示文件图标和名称。
import { IMAGE_EXTENSIONS } from '@/shared/lib/config'
import { cn } from '@/shared/lib/utils'
import * as Tooltip from '@radix-ui/react-tooltip'
import { FileText } from 'lucide-react'

// children 是提示触发区域，url 用于预览，name 提供图片替代文字或普通文件名称。
type Props = {
  url: string
  name: string
  children: React.ReactNode
  className?: string
}

const imageExtensions = IMAGE_EXTENSIONS

// 识别只使用扩展名，不请求内容验证，也不剥离查询串或片段。
// Radix 负责悬停生命周期，内容固定显示在触发区域右侧。
export function AssetPreviewTooltip({ url, name, children, className }: Props) {
  const ext = url.split('.').pop()?.toLowerCase() ?? ''
  const isImage = imageExtensions.includes(ext)

  return (
    <Tooltip.Root>
      <Tooltip.Trigger asChild>
        <div className={cn('inline-block', className)}>{children}</div>
      </Tooltip.Trigger>

      <Tooltip.Content
        side="right"
        align="center"
        className="asset-preview-tooltip__content"
      >
        {isImage ? (
          <img
            src={url}
            alt={name}
            className="preview-image asset-preview-tooltip__image"
          />
        ) : (
          <div className="asset-preview-tooltip__file">
            <FileText size={16} /> {name}
          </div>
        )}
      </Tooltip.Content>
    </Tooltip.Root>
  )
}
