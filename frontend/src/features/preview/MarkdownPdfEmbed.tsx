import { versionAssetSrc } from '@/shared/lib/assetSrc'
import i18next from '@/shared/lib/i18n'
import { useIsMobile } from '@/shared/lib/useIsMobile'
import { ExternalLink, FileText } from 'lucide-react'
import { useMemo } from 'react'

type MarkdownPdfEmbedProps = React.ImgHTMLAttributes<HTMLImageElement> & {
  resolveAssetUrl?: (src: string) => string
}

// source 中的 #page=N 片段保留在最终地址中，浏览器内置 PDF 查看器可据此打开指定页。
function isSameOrigin(src: string): boolean {
  // 相对地址按当前页面来源解析，解析失败时不将资源视为同源。
  try {
    return new URL(src, location.origin).origin === location.origin
  } catch {
    return false
  }
}

// 取出查询和片段之前的末段路径并尝试百分号解码，解码失败时保留原字符。
function fileNameOf(src: string): string {
  const path = src.split(/[?#]/)[0]
  const name = path.slice(path.lastIndexOf('/') + 1)
  try {
    return decodeURIComponent(name)
  } catch {
    return name
  }
}

export function MarkdownPdfEmbed({
  src = '',
  alt,
  width,
  resolveAssetUrl,
}: MarkdownPdfEmbedProps) {
  // 使用与普通图片相同的资源解析和版本参数处理，移动端不渲染内嵌框架。
  const resolvedSrc = useMemo(
    () => resolveAssetUrl?.(src) ?? src,
    [resolveAssetUrl, src],
  )
  const versionedSrc = useMemo(
    () => versionAssetSrc(resolvedSrc),
    [resolvedSrc],
  )
  const isMobile = useIsMobile()
  const openLabel = i18next.t('imagePreview.openInNewTab', { ns: 'viewer' })

  // 移动设备或跨来源 PDF 仅显示新标签页入口，桌面同源资源使用延迟加载的 iframe。
  if (isMobile || !isSameOrigin(versionedSrc)) {
    const fileName = fileNameOf(resolvedSrc)
    const title = alt || fileName
    return (
      <span
        className="markdown-pdf-embed"
        style={width ? { width } : undefined}
      >
        <a
          href={versionedSrc}
          target="_blank"
          rel="noopener noreferrer"
          className="markdown-pdf-card"
        >
          <FileText size={20} className="markdown-pdf-card__icon" />
          <span className="markdown-pdf-card__body">
            <span className="markdown-pdf-card__title">{title}</span>
            <span className="markdown-pdf-card__meta">
              {alt && fileName ? `${fileName} · ` : ''}
              {openLabel}
            </span>
          </span>
          <ExternalLink
            size={14}
            className="markdown-pdf-card__arrow ml-auto"
          />
        </a>
      </span>
    )
  }

  return (
    <span className="markdown-pdf-embed" style={{ width: width || '100%' }}>
      <iframe
        src={versionedSrc}
        title={alt || i18next.t('pdfEmbed.defaultTitle', { ns: 'viewer' })}
        className="markdown-pdf-embed__frame"
        loading="lazy"
      />
      <a
        href={versionedSrc}
        target="_blank"
        rel="noopener noreferrer"
        className="markdown-pdf-embed__open-link"
      >
        <ExternalLink size={14} />
        {openLabel}
      </a>
    </span>
  )
}
