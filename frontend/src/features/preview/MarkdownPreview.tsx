import { TableCellContent } from '@/shared/components/SettingsTable'
import { TooltipProvider } from '@/shared/components/ui/tooltip'
import { useResolvedDesignMode } from '@/features/designtoggle/useResolvedDesignMode'
import { useTranslation } from 'react-i18next'
import { normalizeAssetSrc } from '@/shared/lib/assetSrc'
import {
  buildWikiTitleIndex,
  preprocessWikilinks,
} from '@/shared/lib/preprocessWikilinks'
import { useTreeStore } from '@/features/tree/state/tree'
import 'katex/dist/katex.min.css'
import {
  AnchorHTMLAttributes,
  AudioHTMLAttributes,
  BlockquoteHTMLAttributes,
  Children,
  ClassAttributes,
  type ComponentProps,
  HTMLAttributes,
  ImgHTMLAttributes,
  VideoHTMLAttributes,
  isValidElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import ReactMarkdown from 'react-markdown'
import { JSX } from 'react/jsx-runtime'
import rehypeHighlight from 'rehype-highlight'
import rehypeKatex from 'rehype-katex'
import rehypeRaw from 'rehype-raw'
import rehypeSanitize from 'rehype-sanitize'
import remarkFlexibleMarkers from 'remark-flexible-markers'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import { extractTocEntries } from '@/features/preview/extractTocEntries'
import Headline from '@/features/preview/Headline'
import MarkdownCodeBlock from '@/features/preview/MarkdownCodeBlock'
import MarkdownInlineCode from '@/features/preview/MarkdownInlineCode'
import { MarkdownImage } from '@/features/preview/MarkdownImage'
import { MarkdownLink } from '@/features/preview/MarkdownLink'
import { MarkdownPdfEmbed } from '@/features/preview/MarkdownPdfEmbed'
import '@/features/preview/markdownPreviewCodeTheme.css'
import MermaidBlock from '@/features/preview/MermaidBlock'
import { normalizeMarkdownListIndentation } from '@/features/preview/normalizeMarkdownListIndentation'
import { normalizeMarkdownBlocks } from '@/features/preview/normalizeMarkdownBlocks'
import { rehypeCodeFenceLineNumbers } from '@/features/preview/rehypeCodeFenceLineNumbers'
import { rehypeLineNumber } from '@/features/preview/rehypeLineNumber'
import { rehypeWhitelistStyles } from '@/features/preview/rehypeWhitelistStyles'
import { syntaxHighlightLanguages } from '@/features/preview/syntaxHighlightLanguages'
import { TocDropdownButton } from '@/features/preview/TocDropdownButton'
import { remarkImageSize } from '@/features/preview/remarkImageSize'
import { MarkdownPreviewErrorBoundary } from '@/features/preview/MarkdownPreviewErrorBoundary'
import {
  findScrollParent,
  getSemanticShoutoutTitle,
  getShoutoutConfig,
  isPdfSrc,
  isPlainListParagraph,
  normalizeFootnoteHref,
  transformMarkdownUrl,
  type MarkdownNodeProp,
} from '@/features/preview/previewHelpers'
import { previewSanitizeSchema } from '@/features/preview/previewSanitizeSchema'

type Props = {
  // 要解析并显示的 Markdown 原文，内容变化会触发解析结果重算。
  content: string
  // 当前页面路径供内部链接解析和预览错误边界重置使用。
  path?: string
  // 将文档中的资源地址转换为实际访问地址的页面级解析函数。
  resolveAssetUrl?: (src: string) => string
  // 控制标题是否包裹为可复制锚点链接。
  enableHeadlineLinks?: boolean
  // 控制标题目录数据与目录入口的生成。
  showToc?: boolean
  // 控制目录项是否提供滚动跳转。
  tocClickable?: boolean
  // 目录切换为悬浮状态时通知外层页面。
  onStickyTocChange?: (show: boolean) => void
}

// 组合 Markdown 解析插件、HTML 渲染器和页面内目录，生成预览内容。
export default function MarkdownPreview({
  content,
  path,
  resolveAssetUrl,
  enableHeadlineLinks = true,
  showToc = false,
  tocClickable = true,
  onStickyTocChange,
}: Props) {
  // 当前语言用于提示块标题，设计模式用于 Mermaid 图表的配色。
  const { i18n } = useTranslation('viewer')
  const treeById = useTreeStore((s) => s.byId)
  const resolvedMode = useResolvedDesignMode()

  const markdownLink = useCallback(
    ({
      node,
      ...props
    }: MarkdownNodeProp &
      ClassAttributes<HTMLAnchorElement> &
      AnchorHTMLAttributes<HTMLAnchorElement>) => {
      void node
      return (
        <MarkdownLink
          path={path}
          resolveAssetUrl={resolveAssetUrl}
          {...props}
          href={normalizeFootnoteHref(props.href)}
        />
      )
    },
    [path, resolveAssetUrl],
  )

  const components = useMemo(
    () => ({
      a: markdownLink,
      img: ({
        node,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLImageElement> &
        ImgHTMLAttributes<HTMLImageElement>) => {
        void node
        // PDF 使用嵌入查看器，其他图片由图片组件处理加载和尺寸。
        if (isPdfSrc(props.src)) {
          return (
            <MarkdownPdfEmbed {...props} resolveAssetUrl={resolveAssetUrl} />
          )
        }
        return <MarkdownImage {...props} resolveAssetUrl={resolveAssetUrl} />
      },
      audio: ({
        node,
        ...props
      }: MarkdownNodeProp & AudioHTMLAttributes<HTMLAudioElement>) => {
        void node
        // 音频与视频先解析页面资源地址，再规范化为实际媒体地址。
        const resolvedSrc = resolveAssetUrl?.(props.src ?? '') ?? props.src
        return (
          <audio
            {...props}
            src={resolvedSrc ? normalizeAssetSrc(resolvedSrc) : resolvedSrc}
          />
        )
      },
      video: ({
        node,
        ...props
      }: MarkdownNodeProp & VideoHTMLAttributes<HTMLVideoElement>) => {
        void node
        const resolvedSrc = resolveAssetUrl?.(props.src ?? '') ?? props.src
        return (
          <video
            {...props}
            src={resolvedSrc ? normalizeAssetSrc(resolvedSrc) : resolvedSrc}
          />
        )
      },
      section: ({
        children,
        node,
        className,
        ...props
      }: MarkdownNodeProp &
        HTMLAttributes<HTMLElement> & {
          'data-footnotes'?: boolean | string
        }) => {
        void node
        // 脚注 section 转为带专用样式的 div，普通 section 保留语义标签。
        if ('data-footnotes' in props) {
          return (
            <div
              {...props}
              className={`markdown-footnotes ${className ?? ''}`.trim()}
            >
              {children}
            </div>
          )
        }

        return (
          <section {...props} className={className}>
            {children}
          </section>
        )
      },
      li: ({
        children,
        node,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLLIElement> &
        HTMLAttributes<HTMLLIElement>) => {
        void node
        // 单段落列表项去除多余的段落包装，复杂列表项仍按原子节点输出。
        const childArray = Array.isArray(children) ? children : [children]
        const meaningfulChildren = childArray.filter(
          (child) => child !== null && child !== undefined && child !== false,
        )
        const onlyChild = meaningfulChildren[0]

        if (
          meaningfulChildren.length === 1 &&
          isPlainListParagraph(onlyChild)
        ) {
          return <li {...props}>{onlyChild.props.children}</li>
        }

        return <li {...props}>{children}</li>
      },
      blockquote: ({
        children,
        node,
        className,
        'data-line': dataLine,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLQuoteElement> &
        BlockquoteHTMLAttributes<HTMLQuoteElement> & {
          'data-line'?: string
        }) => {
        void node
        // 识别提示块标记后，将标记段落转换成本地化标题。
        const shoutoutConfig = getShoutoutConfig(children)

        if (!shoutoutConfig) {
          return (
            <blockquote {...props} data-line={dataLine} className={className}>
              {children}
            </blockquote>
          )
        }

        const childArray = Children.toArray(children)
        const markerIndex = childArray.findIndex(
          (child) => isValidElement(child) && child.type === 'p',
        )
        const contentChildren = (
          markerIndex >= 0 ? childArray.slice(markerIndex + 1) : []
        ).filter((child) => typeof child !== 'string' || child.trim() !== '')

        const title = getSemanticShoutoutTitle(
          shoutoutConfig.kind,
          i18n.resolvedLanguage,
        )

        return (
          <aside
            {...props}
            data-line={dataLine}
            className={`markdown-shoutout markdown-shoutout--${shoutoutConfig.kind} ${className ?? ''}`.trim()}
          >
            {title ? <p className="markdown-shoutout__title">{title}</p> : null}
            <div className="markdown-shoutout__content">{contentChildren}</div>
          </aside>
        )
      },
      h1: ({
        children,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLHeadingElement> &
        HTMLAttributes<HTMLHeadingElement>) =>
        enableHeadlineLinks ? (
          <Headline level={1} {...props}>
            {children}
          </Headline>
        ) : (
          <h1 {...props}>{children}</h1>
        ),
      h2: ({
        children,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLHeadingElement> &
        HTMLAttributes<HTMLHeadingElement>) =>
        enableHeadlineLinks ? (
          <Headline level={2} {...props}>
            {children}
          </Headline>
        ) : (
          <h2 {...props}>{children}</h2>
        ),
      h3: ({
        children,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLHeadingElement> &
        HTMLAttributes<HTMLHeadingElement>) =>
        enableHeadlineLinks ? (
          <Headline level={3} {...props}>
            {children}
          </Headline>
        ) : (
          <h3 {...props}>{children}</h3>
        ),
      h4: ({
        children,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLHeadingElement> &
        HTMLAttributes<HTMLHeadingElement>) =>
        enableHeadlineLinks ? (
          <Headline level={4} {...props}>
            {children}
          </Headline>
        ) : (
          <h4 {...props}>{children}</h4>
        ),
      h5: ({
        children,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLHeadingElement> &
        HTMLAttributes<HTMLHeadingElement>) =>
        enableHeadlineLinks ? (
          <Headline level={5} {...props}>
            {children}
          </Headline>
        ) : (
          <h5 {...props}>{children}</h5>
        ),
      h6: ({
        children,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLHeadingElement> &
        HTMLAttributes<HTMLHeadingElement>) =>
        enableHeadlineLinks ? (
          <Headline level={6} {...props}>
            {children}
          </Headline>
        ) : (
          <h6 {...props}>{children}</h6>
        ),
      table: ({
        node,
        ...props
      }: MarkdownNodeProp &
        ClassAttributes<HTMLTableElement> &
        HTMLAttributes<HTMLTableElement>) => {
        void node
        // 表格容器支持横向滚动，单元格提示共用统一的触发延迟。
        return (
          <TooltipProvider delayDuration={300}>
            <div className="table-wrapper">
              <table {...props} className={props.className} />
            </div>
          </TooltipProvider>
        )
      },
      th: ({
        node,
        children,
        ...props
      }: MarkdownNodeProp & ComponentProps<'th'>) => {
        void node
        return (
          <th {...props}>
            <TableCellContent>{children}</TableCellContent>
          </th>
        )
      },
      td: ({
        node,
        children,
        ...props
      }: MarkdownNodeProp & ComponentProps<'td'>) => {
        void node
        return (
          <td {...props}>
            <TableCellContent>{children}</TableCellContent>
          </td>
        )
      },
      pre: MarkdownCodeBlock,
      code: ({
        node,
        ...props
      }: MarkdownNodeProp &
        JSX.IntrinsicAttributes &
        ClassAttributes<HTMLElement> &
        HTMLAttributes<HTMLElement> & {
          'data-line'?: string
          'data-line-numbers'?: string | boolean
        }) => {
        void node
        // Mermaid 代码渲染为图表，其余代码根据语言类名进入代码块或行内组件。
        const { className, children, 'data-line': dataLine } = props
        if (className?.includes('language-mermaid')) {
          const code = String(children ?? '').trim()
          return (
            <MermaidBlock
              code={code}
              dataLine={dataLine}
              theme={resolvedMode === 'dark' ? 'dark' : 'default'}
            />
          )
        }

        if (className?.includes('language-')) {
          return (
            <code data-line={dataLine} className={className} {...props}>
              {children}
            </code>
          )
        }
        if (
          children &&
          typeof children === 'string' &&
          children.includes('\n')
        ) {
          return <code data-line={dataLine}>{children}</code>
        }
        return (
          <MarkdownInlineCode data-line={dataLine} className={className}>
            {children}
          </MarkdownInlineCode>
        )
      },
    }),
    [
      enableHeadlineLinks,
      markdownLink,
      resolveAssetUrl,
      resolvedMode,
      i18n.resolvedLanguage,
    ],
  )

  const wikiTitleIndex = useMemo(
    // 由当前树节点构造标题索引，供 Wiki 链接预处理识别名称和歧义。
    () => buildWikiTitleIndex(treeById),
    [treeById],
  )

  const normalizedContent = useMemo(
    // 预处理顺序为 Wiki 链接解析、提示块转换和列表缩进规范化。
    () =>
      normalizeMarkdownListIndentation(
        normalizeMarkdownBlocks(
          preprocessWikilinks(
            content,
            (title) => wikiTitleIndex.get(title.toLowerCase()) ?? [],
          ),
        ),
      ),
    [content, wikiTitleIndex],
  )

  const tocEntries = useMemo(
    // 目录隐藏时跳过标题扫描，避免计算不会显示的目录数据。
    () => (showToc ? extractTocEntries(normalizedContent) : []),
    [showToc, normalizedContent],
  )

  const inFlowRef = useRef<HTMLDivElement>(null)
  const [showStickyToc, setShowStickyToc] = useState(false)

  useEffect(() => {
    // 目录超过三项时观察文档流锚点，离开视口后通知页面显示悬浮目录。
    if (!showToc || tocEntries.length <= 3) return
    const el = inFlowRef.current
    if (!el) return
    const root = findScrollParent(el)
    const observer = new IntersectionObserver(
      ([entry]) => {
        const sticky = !entry.isIntersecting
        setShowStickyToc(sticky)
        onStickyTocChange?.(sticky)
      },
      { root, threshold: 0 },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [showToc, tocEntries.length, onStickyTocChange])

  // 正文按规范化文本和渲染依赖缓存，树顺序变化不会触发无关的重新解析。
  const markdownBody = useMemo(
    () => (
      <MarkdownPreviewErrorBoundary resetKey={`${path ?? ''}:${content}`}>
        <>
          <ReactMarkdown
            // 单美元符号不作为数学分隔符，避免 Shell 变量和正文文本被误认作公式。
            // 块级数学仍由 remarkMath 解析，行内数学语法不参与转换。
            remarkPlugins={[
              [remarkMath, { singleDollarTextMath: false }],
              remarkGfm,
              remarkFlexibleMarkers,
              remarkImageSize,
            ]}
            rehypePlugins={[
              rehypeRaw,
              rehypeLineNumber,
              rehypeCodeFenceLineNumbers,
              rehypeWhitelistStyles,
              [rehypeKatex, { output: 'html', strict: 'ignore' }],
              [rehypeSanitize, previewSanitizeSchema],
              [
                rehypeHighlight,
                {
                  languages: syntaxHighlightLanguages,
                },
              ],
            ]}
            components={components}
            urlTransform={transformMarkdownUrl}
          >
            {normalizedContent}
          </ReactMarkdown>
          <div id="mermaid-renderer"></div>
        </>
      </MarkdownPreviewErrorBoundary>
    ),
    [normalizedContent, components, path, content],
  )

  if (!showToc || tocEntries.length <= 3) {
    return markdownBody
  }

  return (
    <>
      <div ref={inFlowRef} className="print:hidden">
        {!showStickyToc && (
          <div className="markdown-preview__toc-inline mb-2 flex sm:mb-4">
            <TocDropdownButton entries={tocEntries} clickable={tocClickable} />
          </div>
        )}
      </div>
      {markdownBody}
    </>
  )
}
