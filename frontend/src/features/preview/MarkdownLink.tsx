import { Link } from 'react-router'

import { Button } from '@/shared/components/ui/button'
import { TooltipWrapper } from '@/shared/components/TooltipWrapper'
import { createNavigationVisitState } from '@/shared/lib/navigationVisit'
import {
  DIALOG_CREATE_PAGE_BY_PATH,
  DIALOG_WIKILINK_DISAMBIGUATION,
} from '@/shared/lib/registries/instances'
import {
  buildViewUrl,
  stripBasePath,
  withBasePath,
} from '@/shared/lib/routePath'
import {
  normalizeWikiRoutePath,
  resolveWikiLinkPath,
  toWikiLookupPath,
} from '@/shared/lib/wikiPath'
import { suggestSlug } from '@/shared/lib/api/pages'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTreeStore } from '@/features/tree/state/tree'
import { AnchorHTMLAttributes, ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

// 只读界面用错误色和提示说明链接无对应页面，不提供创建操作。
function BrokenLinkTooltip({
  children,
  label,
}: {
  children: ReactNode
  label: string
}) {
  return (
    <TooltipWrapper label={label} asChild>
      <span className="text-error cursor-default">{children}</span>
    </TooltipWrapper>
  )
}

function safeDecodeURIComponent(s: string): string {
  // 链接预处理器会把标题编码进特殊 href，解码失败时保留原始文字。
  try {
    return decodeURIComponent(s)
  } catch {
    return s
  }
}

function isWebHref(href: string): boolean {
  // http(s) 和协议相对地址交由浏览器按网络链接处理。
  return /^(https?:|\/\/)/i.test(href)
}

function isInternalHref(href: string): boolean {
  // 排除空值、网络地址、邮件链接和页内锚点后，剩余相对路径按 Wiki 路径解析。
  return (
    href !== '' &&
    !isWebHref(href) &&
    !href.startsWith('http') &&
    !href.startsWith('mailto:') &&
    !href.startsWith('#')
  )
}

function isAssetHref(href: string): boolean {
  // 项目资源以 assets/ 开头，可带根路径前缀。
  return href.startsWith('assets/') || href.startsWith('/assets/')
}

// 扩展原生链接属性，补充页面路径和资源地址解析器供预览组件使用。
interface MarkdownLinkProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  href?: string
  children?: ReactNode
  path?: string
  node?: unknown
  resolveAssetUrl?: (src: string) => string
}

export function MarkdownLink({
  href,
  children,
  node,
  resolveAssetUrl,
  ...props
}: MarkdownLinkProps) {
  // 语法树节点仅由 Markdown 组件接口传入，不参与链接渲染。
  void node
  const { t } = useTranslation('viewer')
  const openDialog = useDialogsStore((s) => s.openDialog)
  const getPageByPath = useTreeStore((s) => s.getPageByPath)
  const getPageById = useTreeStore((s) => s.getPageById)

  const editMode = useAppMode() === 'edit'
  const readOnly = useIsReadOnly()

  // 当前路径用于解析相对 Wiki 链接，优先采用调用方提供的文档路径。
  const getCurrentWikiPath = (): string => {
    let locationPath = window.location.pathname
    const stripped = stripBasePath(locationPath)
    if (stripped !== null) {
      locationPath = stripped
    }
    return normalizeWikiRoutePath(props.path ?? buildViewUrl(locationPath))
  }

  // 在任何提前返回之前解析链接并订阅目标页，保证树数据加载或变化时链接状态同步更新。
  let normalizedHref: string | null = null
  if (
    href !== undefined &&
    !href.startsWith('wikilink-ambiguous:') &&
    !href.startsWith('wikilink-notfound:') &&
    isInternalHref(href) &&
    !isAssetHref(href)
  ) {
    // 根路径作为绝对 Wiki 路径，其他路径相对于当前文档所在目录解析。
    normalizedHref = href.startsWith('/')
      ? normalizeWikiRoutePath(href)
      : resolveWikiLinkPath(getCurrentWikiPath(), href)
  }
  // 树索引使用不带开头斜线、查询参数和锚点的路径作为查找键。
  const lookupPath =
    normalizedHref === null ? null : toWikiLookupPath(normalizedHref)
  const targetPage = useTreeStore((s) =>
    lookupPath === null ? null : (s.byPath[lookupPath] ?? null),
  )

  if (href === undefined) {
    // 没有 href 时保留 Markdown 生成的子内容，不创建不可点击的空链接。
    return <>{children}</>
  }

  if (href.startsWith('wikilink-ambiguous:')) {
    // 歧义链接携带编码标题，点击后打开候选页面选择对话框。
    const title = safeDecodeURIComponent(
      href.slice('wikilink-ambiguous:'.length),
    )
    return (
      <Button
        variant="link"
        onClick={() => openDialog(DIALOG_WIKILINK_DISAMBIGUATION, { title })}
        className="text-brand hover:text-brand-dark m-0 p-0 text-base no-underline hover:no-underline"
      >
        {children}
      </Button>
    )
  }

  if (href.startsWith('wikilink-notfound:')) {
    // 未找到的 Wiki 链接在可编辑模式下打开创建页流程，只读模式显示错误提示。
    const title = safeDecodeURIComponent(
      href.slice('wikilink-notfound:'.length),
    )
    if (!readOnly) {
      return (
        <Button
          variant="link"
          onClick={async () => {
            // 新页面放在当前文档所在文件夹，当前页不存在时使用根目录。
            const currentPage = getPageByPath(
              toWikiLookupPath(getCurrentWikiPath()),
            )
            const parentId = currentPage?.parentId ?? ''
            const folderPath = parentId
              ? (getPageById(parentId)?.path ?? '')
              : ''

            // 服务器生成可用路径段，失败时回退到标题文本，再与文件夹路径拼接。
            const slug = await suggestSlug(parentId, title).catch(() => '')
            const initialPath = [folderPath, slug || title]
              .filter(Boolean)
              .join('/')
            openDialog(DIALOG_CREATE_PAGE_BY_PATH, {
              initialPath,
              initialTitle: title,
              forwardToEditMode: !editMode,
            })
          }}
          className="text-error hover:text-error/80 m-0 p-0 text-base no-underline hover:no-underline"
        >
          {children}
        </Button>
      )
    }
    return (
      <BrokenLinkTooltip label={t('markdownPreview.brokenLinkTooltip')}>
        {children}
      </BrokenLinkTooltip>
    )
  }

  const isWebLink = isWebHref(href)
  let isSameOrigin = false
  if (isWebLink) {
    try {
      const url = new URL(href, window.location.href)
      isSameOrigin =
        (url.protocol === 'http:' || url.protocol === 'https:') &&
        url.origin === window.location.origin
    } catch {
      // 无效网络地址不参与同源判断，后续仍按普通外部链接渲染。
    }
  }

  const handleOpenCreatePageDialog = (path: string) => {
    // 已解析的内部路径以只读路径交给创建页对话框，标题由对话框流程处理。
    openDialog(DIALOG_CREATE_PAGE_BY_PATH, {
      initialPath: path,
      readOnlyPath: true,
      forwardToEditMode: !editMode,
    })
  }

  if (isInternalHref(href) && isAssetHref(href)) {
    // 资源链接先统一成 /assets/ 路径，再通过页面资源解析器和部署前缀生成 href。
    const path = href.startsWith('/assets/')
      ? href
      : '/assets/' + href.slice('assets/'.length)

    const resolvedPath = resolveAssetUrl?.(path) ?? path
    const assetHref = withBasePath(resolvedPath)
    return (
      <a
        href={assetHref}
        {...props}
        target="_blank"
        rel="noopener noreferrer"
        className="text-brand hover:text-brand-dark no-underline hover:underline"
      >
        {children}
      </a>
    )
  }

  if (normalizedHref !== null && lookupPath !== null) {
    // 内部页面按规范化路径查询树，未命中时在可编辑模式提供创建入口。
    const normalizedTargetPath = lookupPath
    const pageExists = !!targetPage
    if (!pageExists && !readOnly) {
      return (
        <Button
          variant="link"
          onClick={() => {
            handleOpenCreatePageDialog(normalizedTargetPath)
          }}
          className="text-error hover:text-error/80 m-0 p-0 text-base no-underline hover:no-underline"
        >
          {children}
        </Button>
      )
    }

    if (!pageExists) {
      // 只读模式不提供创建按钮，只显示不可用链接的提示样式。
      return (
        <BrokenLinkTooltip label={t('markdownPreview.brokenLinkTooltip')}>
          {children}
        </BrokenLinkTooltip>
      )
    }

    return (
      // 已存在的内部页面使用路由链接，并附带导航访问状态供最近访问记录使用。
      <Link
        to={normalizedHref}
        state={createNavigationVisitState()}
        {...props}
        className="no-underline hover:underline"
      >
        {children}
      </Link>
    )
  }

  return (
    // 页内锚点和同源网络地址留在当前窗口，其他链接在新窗口打开并隔离 opener。
    <a
      href={href}
      {...props}
      target={isSameOrigin || href.startsWith('#') ? undefined : '_blank'}
      rel="noopener noreferrer"
      className="text-brand hover:text-brand-dark no-underline hover:underline"
    >
      {children}
    </a>
  )
}
