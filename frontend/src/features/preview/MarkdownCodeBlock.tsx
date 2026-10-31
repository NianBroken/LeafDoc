import { TooltipWrapper } from '@/shared/components/TooltipWrapper'
import { Button } from '@/shared/components/ui/button'
import { cn } from '@/shared/lib/utils'
import { Check, Copy } from 'lucide-react'
import {
  ClassAttributes,
  HTMLAttributes,
  ReactNode,
  isValidElement,
} from 'react'
import { useTranslation } from 'react-i18next'
import { splitHighlightedLines } from '@/features/preview/splitHighlightedLines'
import { readTextContent, useCodeCopy } from '@/features/preview/useCodeCopy'

type CodeElementProps = {
  className?: string
  children?: ReactNode
  'data-line-numbers'?: string | boolean
}

// 识别代码围栏转换器写入的行号标记，也兼容 class 中的旧式标记。
function hasLineNumbers(props: CodeElementProps): boolean {
  const flag = props['data-line-numbers']
  if (flag === true || flag === 'true' || flag === '') {
    return true
  }
  return (props.className ?? '').split(/\s+/).some((cls) => cls.endsWith('='))
}

// 将代码节点包装为带复制按钮的代码块，并按需把高亮内容拆成逐行结构。
export default function MarkdownCodeBlock(
  props: ClassAttributes<HTMLPreElement> &
    HTMLAttributes<HTMLPreElement> & { children?: ReactNode; node?: unknown },
) {
  const { t } = useTranslation('viewer')
  const { children, node, ...preProps } = props
  // node 由 Markdown 组件接口传入，此处的展示只依赖实际渲染出的子节点。
  void node
  const child = Array.isArray(children) ? children[0] : children
  const childIsCodeElement = isValidElement<CodeElementProps>(child)
  const className = childIsCodeElement ? (child.props.className ?? '') : ''
  const code = childIsCodeElement ? readTextContent(child.props.children) : ''
  const { copied, copyCode } = useCodeCopy(code)

  if (!childIsCodeElement) {
    // 普通 pre 内容不符合代码节点结构，保留原始 HTML 渲染形态。
    return <pre {...preProps}>{children}</pre>
  }

  const showLineNumbers = hasLineNumbers(child.props)

  const isCodeBlock = className.includes('language-') || code.includes('\n')
  if (!isCodeBlock) {
    // 单行且没有语言标记的内容按普通 pre 展示，不增加代码块工具栏。
    return <pre {...preProps}>{children}</pre>
  }

  const highlightedChildren = child.props.children
  const lines = showLineNumbers
    ? splitHighlightedLines(highlightedChildren)
    : null

  return (
    <div
      className={cn(
        'markdown-code-block',
        showLineNumbers && 'markdown-code-block--line-numbers',
      )}
    >
      <div className="markdown-code-block__actions">
        <TooltipWrapper
          label={
            copied ? t('codeBlock.copiedTooltip') : t('codeBlock.copyTooltip')
          }
        >
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="markdown-code-block__copy-button"
            onClick={copyCode}
            aria-label={
              copied
                ? t('codeBlock.copiedAriaLabel')
                : t('codeBlock.copyAriaLabel')
            }
            data-testid="markdown-code-copy-button"
          >
            {copied ? <Check /> : <Copy />}
          </Button>
        </TooltipWrapper>
      </div>
      <pre
        {...preProps}
        className={`custom-scrollbar ${preProps.className ?? ''}`.trim()}
      >
        {showLineNumbers && lines ? (
          <code className={className} data-line-numbers="true">
            <span className="markdown-code-block__lines">
              {lines.map((line, index) => (
                <span
                  key={`code-line-${index + 1}`}
                  className="markdown-code-block__line"
                >
                  <span
                    className="markdown-code-block__line-number"
                    aria-hidden="true"
                  >
                    {index + 1}
                  </span>
                  <span className="markdown-code-block__line-content">
                    {line.length > 0 ? line : '\n'}
                  </span>
                </span>
              ))}
            </span>
          </code>
        ) : (
          children
        )}
      </pre>
    </div>
  )
}
