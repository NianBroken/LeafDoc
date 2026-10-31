// 展示搜索命中项的标题、摘要、种类和路径，并把锚点引用交给列表处理键盘定位。
import i18next from '@/shared/lib/i18n'
import { SearchResultItem } from '@/shared/lib/api/search'
import { createNavigationVisitState } from '@/shared/lib/navigationVisit'
import { buildViewUrl } from '@/shared/lib/routePath'
import { normalizeWikiRoutePath } from '@/shared/lib/wikiPath'
import { forwardRef } from 'react'
import { Link, useLocation } from 'react-router'
import { usePageEditorStore } from '@/features/editor/pageEditorStore'
import HighlightedSearchTitle from '@/features/search/HighlightedSearchTitle'

const t = (key: string, opts?: Record<string, unknown>) =>
  i18next.t(key, { ...opts, ns: 'search' })

type SearchResultCardProps = {
  item: SearchResultItem
  isSelected?: boolean
  onMouseEnter?: () => void
  onFocus?: () => void
}

const SearchResultCard = forwardRef<HTMLAnchorElement, SearchResultCardProps>(
  function SearchResultCard(
    { item, isSelected = false, onMouseEnter, onFocus },
    ref,
  ) {
    const location = useLocation()
    const currentEditorPageId = usePageEditorStore(
      // 优先使用当前编辑页面，没有时使用初始页面 ID，使编辑中的搜索结果也能突出显示。
      (state) => state.page?.id ?? state.initialPage?.id,
    )
    const currentViewPath = normalizeWikiRoutePath(
      buildViewUrl(location.pathname),
    )
    const resultPath = normalizeWikiRoutePath(item.path)
    const resultUrl = `${resultPath}${location.search}`
    // 路由匹配、编辑页面匹配和键盘选中任一成立即显示选中样式，只有路由匹配设置 aria-current。
    const isRouteActive = currentViewPath === resultPath
    const isEditorActive = currentEditorPageId === item.page_id
    const isActive = isRouteActive || isEditorActive || isSelected
    const kindLabel =
      item.kind === 'section'
        ? t('resultCard.kindSection')
        : t('resultCard.kindPage')

    return (
      <Link
        // 保留现有查询参数，导航状态使用访问标记，使页面打开流程能够区分访问事件。
        ref={ref}
        to={resultUrl}
        state={createNavigationVisitState()}
        data-testid={`search-result-card-${item.page_id}`}
        aria-current={isRouteActive ? 'page' : undefined}
        onMouseEnter={onMouseEnter}
        onFocus={onFocus}
        className={`list-view__item search-result-card ${
          isActive ? 'list-view__item--active search-result-card--selected' : ''
        } ${isRouteActive ? 'search-result-card--route-active' : ''}`.trim()}
      >
        <div
          className="search-result-card__title"
          title={item.title}
          data-testid={`search-result-card-title-${item.page_id}`}
        >
          <HighlightedSearchTitle text={item.title} />
        </div>
        <div className="search-result-card__excerpt">
          <HighlightedSearchTitle text={item.excerpt} />
        </div>
        <div className="search-result-card__meta">
          <span className="search-result-card__badge">{kindLabel}</span>
        </div>
        <div className="search-result-card__path">
          {/* 路径分隔符两侧加入空格仅用于展示，不改变链接地址。 */}
          {item.path.split('/').join(' / ')}
        </div>
      </Link>
    )
  },
)

export default SearchResultCard
