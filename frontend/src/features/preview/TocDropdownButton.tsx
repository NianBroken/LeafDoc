import { Button } from '@/shared/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/components/ui/dropdown-menu'
import type { PageAttachment } from '@/shared/lib/api/assets'
import { withBasePath } from '@/shared/lib/routePath'
import { scrollToHeadlineHash } from '@/shared/lib/scrollToHeadline'
import { cn } from '@/shared/lib/utils'
import { ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { TocEntry } from '@/features/preview/extractTocEntries'
import { tocTitleKey } from '@/features/preview/tocTitleKey'
import { useTocScrollSpy } from '@/features/preview/useTocScrollSpy'

// 提供标题目录和附件下载的下拉入口，标题点击通过共享滚动工具定位页面锚点。
type Props = {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  entries: TocEntry[]
  clickable?: boolean
  activeId?: string | null
  downloads?: PageAttachment[]
}

function getTocEntryClassName(level: number) {
  // 按标题级别缩进，超过三级的输入也使用最深层样式。
  if (level <= 1) {
    return 'text-sm font-semibold'
  }

  if (level === 2) {
    return 'pl-6 text-sm font-medium text-foreground/90'
  }

  if (level === 3) {
    return 'pl-10 text-sm'
  }

  return 'pl-12 text-sm'
}

export function TocDropdownButton({
  open,
  onOpenChange,
  entries,
  clickable = true,
  activeId: externalActiveId,
  downloads = [],
}: Props) {
  const { t } = useTranslation('viewer')
  // 外部提供活动标题时不重复安装滚动监听，undefined 才表示由组件自行追踪。
  const internalActiveId = useTocScrollSpy(
    externalActiveId !== undefined ? [] : entries,
  )
  const activeId =
    externalActiveId !== undefined ? externalActiveId : internalActiveId

  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          {t(tocTitleKey(entries.length, downloads.length))}
          <ChevronDown size={12} />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="max-h-[70vh] max-w-96 min-w-48 overflow-y-auto"
      >
        {entries.map((entry) => (
          <DropdownMenuItem
            key={entry.id}
            className={cn(
              getTocEntryClassName(entry.level),
              activeId === entry.id && 'text-brand',
              clickable && 'cursor-pointer',
            )}
            onSelect={
              // 禁用点击时不注册选择回调，目录项仍保留展示和当前项样式。
              clickable
                ? () => {
                    scrollToHeadlineHash(`#${encodeURIComponent(entry.id)}`, {
                      waitForStableLayout: false,
                    })
                  }
                : undefined
            }
          >
            {entry.text}
          </DropdownMenuItem>
        ))}
        {downloads.length > 0 && (
          // 无标题时下载项不显示分隔线与额外标题，有标题时才单独标记附件区域。
          <>
            {entries.length > 0 && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuLabel>{t('toc.downloads')}</DropdownMenuLabel>
              </>
            )}
            {downloads.map((file) => (
              <DropdownMenuItem key={file.url} asChild>
                <a
                  href={withBasePath(file.url)}
                  download={file.name}
                  className="cursor-pointer text-sm"
                  title={file.name}
                  data-testid={`toc-dropdown-download-${file.name}`}
                >
                  {file.name}
                </a>
              </DropdownMenuItem>
            ))}
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
