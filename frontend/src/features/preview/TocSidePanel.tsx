import { scrollToHeadlineHash } from '@/shared/lib/scrollToHeadline'
import { withBasePath } from '@/shared/lib/routePath'
import { cn } from '@/shared/lib/utils'
import { useTocPanelStore } from '@/features/preview/state/tocPanel'
import { PanelRightClose, PanelRightOpen } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { PageAttachment } from '@/shared/lib/api/assets'
import type { TocEntry } from '@/features/preview/extractTocEntries'
import { tocTitleKey } from '@/features/preview/tocTitleKey'
import { useTocScrollSpy } from '@/features/preview/useTocScrollSpy'

type Props = {
  // 解析后的标题项提供锚点、层级和显示文字，分别用于跳转、缩进和标签。
  entries: TocEntry[]
  // 有值时由父级提供当前标题，缺省时由本组件根据滚动位置计算。
  activeId?: string | null
  // 附件只作为下载链接显示，不进入标题滚动监听。
  downloads?: PageAttachment[]
}

function getIndentClass(level: number): string {
  // 一级标题与列表左侧对齐，二级及更深标题使用两档固定缩进。
  if (level <= 1) return ''
  if (level === 2) return 'pl-3'
  return 'pl-6'
}

export function TocSidePanel({
  entries,
  activeId: externalActiveId,
  downloads = [],
}: Props) {
  const { t } = useTranslation('viewer')
  // 折叠标志存放在共享状态中，组件重渲染时沿用同一值。
  const collapsed = useTocPanelStore((state) => state.collapsed)
  const setCollapsed = useTocPanelStore((state) => state.setCollapsed)
  // 父级提供活动标题时交给父级维护滚动监听，避免重复订阅同一批标题。
  const internalActiveId = useTocScrollSpy(
    externalActiveId !== undefined ? [] : entries,
  )
  const activeId =
    externalActiveId !== undefined ? externalActiveId : internalActiveId

  // 折叠和展开共用同一个按钮节点，只切换图标、标签和列表可见状态。
  // 面板宽度由样式固定，折叠目录不会推动阅读正文所在的主列。
  return (
    <nav
      className={cn(
        'page-viewer__toc-panel',
        collapsed && 'page-viewer__toc-panel--collapsed',
      )}
      aria-label={t('toc.title')}
      data-testid="toc-side-panel"
    >
      <div className="page-viewer__toc-panel-header">
        <p
          className={cn(
            'page-viewer__toc-panel-title',
            collapsed && 'page-viewer__toc-panel-hidden',
          )}
          aria-hidden={collapsed}
        >
          {t(tocTitleKey(entries.length, downloads.length))}
        </p>
        <button
          type="button"
          className="page-viewer__toc-panel-toggle-button"
          onClick={() => setCollapsed(!collapsed)}
          title={collapsed ? t('toc.expand') : t('toc.collapse')}
          aria-label={collapsed ? t('toc.expand') : t('toc.collapse')}
          data-testid={
            collapsed ? 'toc-side-panel-expand' : 'toc-side-panel-collapse'
          }
        >
          {collapsed ? (
            <PanelRightOpen size={14} />
          ) : (
            <PanelRightClose size={14} />
          )}
        </button>
      </div>
      <ul
        className={cn(
          'page-viewer__toc-panel-list',
          'custom-scrollbar',
          collapsed && 'page-viewer__toc-panel-hidden',
        )}
        aria-hidden={collapsed}
        data-testid="toc-side-panel-list"
      >
        {/* 标题项点击后把锚点编码并滚动到页面中对应的标题元素。 */}
        {entries.map((entry) => (
          <li key={entry.id}>
            <button
              className={cn(
                'page-viewer__toc-panel-entry',
                getIndentClass(entry.level),
                activeId === entry.id && 'page-viewer__toc-panel-entry--active',
              )}
              onClick={() =>
                scrollToHeadlineHash(`#${encodeURIComponent(entry.id)}`, {
                  waitForStableLayout: false,
                })
              }
              title={entry.text}
              data-testid={`toc-entry-${entry.id}`}
              tabIndex={collapsed ? -1 : 0}
            >
              {entry.text}
            </button>
          </li>
        ))}
        {downloads.length > 0 && (
          <>
            {/* 仅当标题目录与附件并存时显示附件分组标题。 */}
            {entries.length > 0 && (
              <li>
                <p
                  className={cn(
                    'page-viewer__toc-panel-title mt-3',
                    collapsed && 'page-viewer__toc-panel-hidden',
                  )}
                  aria-hidden={collapsed}
                >
                  {t('toc.downloads')}
                </p>
              </li>
            )}
            {downloads.map((file) => (
              <li key={file.url}>
                <a
                  href={withBasePath(file.url)}
                  download={file.name}
                  className="page-viewer__toc-panel-entry"
                  title={file.name}
                  data-testid={`toc-download-${file.name}`}
                  tabIndex={collapsed ? -1 : 0}
                >
                  {file.name}
                </a>
              </li>
            ))}
          </>
        )}
      </ul>
    </nav>
  )
}
