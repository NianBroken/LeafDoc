import { TooltipWrapper } from '@/shared/components/TooltipWrapper'
import { Button } from '@/shared/components/ui/button'
import { useIsMobile } from '@/shared/lib/useIsMobile'
import { Check, Copy } from 'lucide-react'
import { ClassAttributes, HTMLAttributes, MouseEvent, ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { readTextContent, useCodeCopy } from '@/features/preview/useCodeCopy'

type Props = ClassAttributes<HTMLElement> &
  HTMLAttributes<HTMLElement> & { children?: ReactNode }

// 行内代码保留 Markdown 渲染器传入的属性，并在桌面端提供复制入口。
export default function MarkdownInlineCode({
  children,
  className,
  ...codeProps
}: Props) {
  const { t } = useTranslation('viewer')
  const isMobile = useIsMobile()
  const code = readTextContent(children)
  // 复制状态由共享 Hook 管理，按钮提示与图标据此同步变化。
  const { copied, copyCode } = useCodeCopy(code)

  const handleCopy = (event: MouseEvent<HTMLButtonElement>) => {
    // 阻止按钮点击继续触发行内代码所在链接或其他父级交互。
    event.preventDefault()
    event.stopPropagation()
    copyCode()
  }

  return (
    <span className="markdown-inline-code">
      <code {...codeProps} className={`inline-code ${className ?? ''}`.trim()}>
        {children}
      </code>
      {!isMobile && (
        <TooltipWrapper
          label={
            copied ? t('codeBlock.copiedTooltip') : t('codeBlock.copyTooltip')
          }
          asChild
        >
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="markdown-inline-code__copy-button"
            onClick={handleCopy}
            aria-label={
              copied
                ? t('codeBlock.copiedAriaLabel')
                : t('codeBlock.copyAriaLabel')
            }
            data-testid="markdown-inline-code-copy-button"
          >
            {copied ? <Check /> : <Copy />}
          </Button>
        </TooltipWrapper>
      )}
    </span>
  )
}
