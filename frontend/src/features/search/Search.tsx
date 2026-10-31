// 搜索侧栏将查询词与标签同步到地址栏，延迟请求分页结果，并支持标签筛选与键盘打开。
import i18next from '@/shared/lib/i18n'
import { Trans } from 'react-i18next'
import {
  ListView,
  ListViewList,
  ListViewStatus,
} from '@/shared/components/ListView'
import { Pagination } from '@/shared/components/Pagination'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/shared/components/ui/accordion'
import { Input, InputGroup } from '@/shared/components/ui/input'
import {
  searchPages,
  SearchResultItem,
  SearchTagFacet,
} from '@/shared/lib/api/search'
import { deferStateUpdate } from '@/shared/lib/deferState'
import { normalizeWikiRoutePath } from '@/shared/lib/wikiPath'
import { fetchTags, TagCount } from '@/shared/lib/api/tags'
import { useDebounce } from '@/shared/lib/useDebounce'
import { getShortcutDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { useSidebarStore } from '@/features/sidebar/state/sidebar'
import { X } from 'lucide-react'
import {
  startTransition,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import { useLocation, useNavigate, useSearchParams } from 'react-router'
import SearchResultCard from '@/features/search/SearchResultCard'

type SearchProps = {
  active?: boolean
}

const t = (key: string, opts?: Record<string, unknown>) =>
  i18next.t(key, { ...opts, ns: 'search' })

// 输入框聚焦时仅放行目录与搜索面板切换快捷键，组合键从统一目录读取。
const SEARCH_INPUT_ALLOWED_HOTKEYS = [
  getShortcutDefinition('sidebar.explorer.open').keyCombo,
  getShortcutDefinition('sidebar.search.open').keyCombo,
].join(' ')

export default function Search({ active = false }: SearchProps) {
  const location = useLocation()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const urlQuery = searchParams.get('q') ?? ''
  const [inputQuery, setInputQuery] = useState(urlQuery)
  const activeTags = searchParams.getAll('tags')
  const activeTagsKey = activeTags.join('\n')
  // 标签列表转换为字符串参与延迟状态，查询词使用独立的较长延迟。
  const debouncedActiveTagsKey = useDebounce(activeTagsKey, 180)

  const [loading, setLoading] = useState(
    () => urlQuery.length >= 3 || activeTags.length > 0,
  )
  const [results, setResults] = useState<SearchResultItem[]>([])
  const [totalCount, setTotalCount] = useState(0)
  const [page, setPage] = useState(0)
  const [activeIndex, setActiveIndex] = useState(0)
  const [availableTags, setAvailableTags] = useState<TagCount[]>([])
  const [availableTagsError, setAvailableTagsError] = useState(false)
  const [facetTags, setFacetTags] = useState<SearchTagFacet[]>([])
  const searchInputRef = useRef<HTMLInputElement | null>(null)
  const searchFocusRequestId = useSidebarStore(
    (state) => state.searchFocusRequestId,
  )
  const resultRefs = useRef<(HTMLAnchorElement | null)[]>([])
  const latestRequestIdRef = useRef(0)

  const limit = 10
  const debouncedQuery = useDebounce(inputQuery, 300)
  const debouncedActiveTags = useMemo(
    () =>
      debouncedActiveTagsKey === '' ? [] : debouncedActiveTagsKey.split('\n'),
    [debouncedActiveTagsKey],
  )
  const hasSearchQuery = debouncedQuery.length >= 3
  // 请求门槛按未去空白的查询长度判断，界面空闲状态则按去空白文本与当前标签判断。
  const trimmedQuery = inputQuery.trim()
  const activeQueryLabel = hasSearchQuery ? debouncedQuery : trimmedQuery
  const isIdleMode = trimmedQuery === '' && activeTags.length === 0
  const hasImmediateFilters = !isIdleMode
  const hasDebouncedFilters = hasSearchQuery || debouncedActiveTags.length > 0
  const hasResults = results.length > 0
  const clampedActiveIndex =
    results.length === 0 ? 0 : Math.min(activeIndex, results.length - 1)
  const visibleTags = isIdleMode ? availableTags : facetTags
  // 空闲时显示标签接口结果，有筛选时显示搜索响应中的标签分面。
  const availableTagsLabel =
    activeTags.length === 0
      ? t('tags.available', { count: visibleTags.length })
      : t('tags.selected', { count: activeTags.length })
  const activeFilterLabel = useMemo(() => {
    if (activeQueryLabel && activeTags.length > 0) {
      return t('results.filterQueryWithTags', {
        query: activeQueryLabel,
        count: activeTags.length,
      })
    }
    if (activeQueryLabel) {
      return t('results.filterQuery', { query: activeQueryLabel })
    }
    if (activeTags.length === 1) {
      return t('results.filterTagged', { tag: activeTags[0] })
    }
    return t('results.filterTaggedWith', { count: activeTags.length })
  }, [activeQueryLabel, activeTags])

  const invalidatePendingRequests = () => {
    // 递增序号使旧请求的完成回调失效，此动作不取消实际网络连接。
    latestRequestIdRef.current += 1
  }

  const toggleActiveTag = useCallback(
    (tag: string) => {
      // 更新重复 tags 参数并保留其他参数；选择变化后回到首页和第一个结果。
      invalidatePendingRequests()
      setLoading(true)
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          const current = next.getAll('tags')
          next.delete('tags')
          if (current.includes(tag)) {
            current
              .filter((existingTag) => existingTag !== tag)
              .forEach((existingTag) => next.append('tags', existingTag))
          } else {
            ;[...current, tag].forEach((existingTag) =>
              next.append('tags', existingTag),
            )
          }
          return next
        },
        { replace: true },
      )
      setPage(0)
      setActiveIndex(0)
    },
    [setSearchParams],
  )

  const clearActiveTags = useCallback(() => {
    // 只移除标签参数，保留查询词；短查询下同时清空结果和加载状态。
    invalidatePendingRequests()
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        next.delete('tags')
        return next
      },
      { replace: true },
    )
    setPage(0)
    setActiveIndex(0)
    if (urlQuery.trim().length < 3) {
      setResults([])
      setTotalCount(0)
      setFacetTags([])
      setLoading(false)
    }
  }, [setSearchParams, urlQuery])

  useEffect(() => {
    if (active) {
      searchInputRef.current?.focus()
    }
    // 面板已打开时再次按搜索快捷键会改变聚焦请求序号，从而重新聚焦输入框。
  }, [active, searchFocusRequestId])

  useEffect(() => {
    if (!active) {
      return
    }

    deferStateUpdate(() => {
      // 活动面板响应地址栏查询变化，延后更新本地输入并避免相同值重复写入。
      setInputQuery((currentQuery) =>
        currentQuery === urlQuery ? currentQuery : urlQuery,
      )
    })
  }, [active, urlQuery])

  useEffect(() => {
    if (!active) {
      return
    }

    let cancelled = false

    fetchTags('', 200)
      // 获取最多 200 个可用标签；清理标记只阻止写入结果，不取消已发出的请求。
      .then((tags) => {
        if (cancelled) return
        setAvailableTags(tags)
      })
      .catch(() => {
        if (cancelled) return
        setAvailableTags([])
        setAvailableTagsError(true)
      })

    return () => {
      cancelled = true
    }
  }, [active])

  useEffect(() => {
    resultRefs.current = resultRefs.current.slice(0, results.length)
  }, [results.length])

  useEffect(() => {
    if (!hasResults) return

    resultRefs.current[clampedActiveIndex]?.scrollIntoView({
      block: 'nearest',
    })
  }, [clampedActiveIndex, hasResults])

  useEffect(() => {
    if (!hasDebouncedFilters) {
      return
    }

    const requestId = latestRequestIdRef.current + 1
    // 请求仅受延迟筛选和分页依赖控制，不以面板 active 状态作为发起条件。
    latestRequestIdRef.current = requestId

    searchPages(
      hasSearchQuery ? debouncedQuery : '',
      page * limit,
      limit,
      debouncedActiveTags,
    )
      .then((data) => {
        // 只接受最新序号的结果，缺少命中列表或分面时分别使用空数组。
        if (latestRequestIdRef.current !== requestId) return
        setResults(data.items || [])
        setTotalCount(data.count)
        setFacetTags(data.tag_facets || [])
      })
      .catch((err) => {
        // 最新请求失败时记录错误并清空结果，过期请求失败不会覆盖当前列表。
        if (latestRequestIdRef.current !== requestId) return
        console.error('Search failed', err)
        setResults([])
        setTotalCount(0)
        setFacetTags([])
      })
      .finally(() => {
        if (latestRequestIdRef.current !== requestId) return
        setLoading(false)
      })
  }, [
    debouncedActiveTags,
    debouncedQuery,
    hasDebouncedFilters,
    hasSearchQuery,
    page,
  ])

  const clearSearch = () => {
    // 清除查询词但保留标签，仍有标签时等待纯标签搜索，否则直接恢复空列表。
    invalidatePendingRequests()
    setInputQuery('')
    startTransition(() => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          next.delete('q')
          return next
        },
        { replace: true },
      )
    })
    setPage(0)
    setActiveIndex(0)
    if (activeTags.length === 0) {
      setResults([])
      setTotalCount(0)
      setFacetTags([])
      setLoading(false)
      return
    }

    setLoading(true)
  }

  const openActiveResult = () => {
    // 按当前结果数量限制的索引取命中项，保留地址栏查询参数进行站内导航。
    const activeResult = results[clampedActiveIndex]
    if (!activeResult) return

    navigate({
      pathname: normalizeWikiRoutePath(activeResult.path),
      search: location.search,
    })
  }

  return (
    <div className="search">
      <InputGroup className="search__input-wrapper">
        <Input
          ref={searchInputRef}
          autoFocus
          type="text"
          placeholder={t('input.placeholder')}
          value={inputQuery}
          data-testid="search-input"
          data-allow-hotkeys={SEARCH_INPUT_ALLOWED_HOTKEYS}
          onChange={(e) => {
            // 输入立即更新，地址参数用替换方式写入历史，实际搜索等待延迟值变化。
            const nextQuery = e.target.value
            invalidatePendingRequests()
            setInputQuery(nextQuery)
            startTransition(() => {
              setSearchParams(
                (prev) => {
                  const next = new URLSearchParams(prev)
                  if (nextQuery) {
                    next.set('q', nextQuery)
                  } else {
                    next.delete('q')
                  }
                  return next
                },
                { replace: true },
              )
            })
            setPage(0)
            setActiveIndex(0)

            if (nextQuery.length >= 3 || activeTags.length > 0) {
              setLoading(true)
            } else {
              setResults([])
              setTotalCount(0)
              setFacetTags([])
              setLoading(false)
            }
          }}
          onKeyDown={(e) => {
            // 上下箭头在现有结果范围内移动，回车打开当前项；没有结果时保留默认行为。
            if (e.key === 'ArrowDown') {
              if (!hasResults) return

              e.preventDefault()
              setActiveIndex((current) =>
                Math.min(current + 1, Math.max(results.length - 1, 0)),
              )
            }

            if (e.key === 'ArrowUp') {
              if (!hasResults) return

              e.preventDefault()
              setActiveIndex((current) => Math.max(current - 1, 0))
            }

            if (e.key === 'Enter') {
              if (!hasResults) return

              e.preventDefault()
              openActiveResult()
            }
          }}
          className="search__input"
        />
        {inputQuery && (
          <button
            type="button"
            onClick={clearSearch}
            className="search__clear-button"
            title={t('input.clearButton')}
            data-testid="search-clear-button"
          >
            <X size={16} />
          </button>
        )}
      </InputGroup>

      {(visibleTags.length > 0 || availableTagsError) && (
        <Accordion
          type="single"
          collapsible
          className="browse-tags__accordion"
          data-testid="search-tags-accordion"
        >
          <AccordionItem
            value="all-tags"
            className="browse-tags__accordion-item"
          >
            <AccordionTrigger
              className="browse-tags__accordion-trigger"
              data-testid="search-tags-accordion-trigger"
            >
              <span className="browse-tags__accordion-title">
                {t('tags.accordionTitle')}
              </span>
              <span className="browse-tags__accordion-summary">
                {availableTagsLabel}
              </span>
            </AccordionTrigger>
            <AccordionContent className="browse-tags__accordion-content">
              {availableTagsError ? (
                <p
                  className="browse-tags__accordion-empty"
                  data-testid="tags-available-error"
                >
                  {t('tags.error')}
                </p>
              ) : (
                <div
                  className="browse-tags__tag-list"
                  data-testid="search-tags-list"
                >
                  {visibleTags.map(({ tag, count }) => {
                    const isActive = activeTags.includes(tag)
                    return (
                      <button
                        key={tag}
                        type="button"
                        className={`browse-tags__tag-filter ${isActive ? 'browse-tags__tag-filter--active' : ''}`.trim()}
                        onClick={() => toggleActiveTag(tag)}
                        aria-pressed={isActive}
                        data-testid={`tags-filter-${tag}`}
                      >
                        <span
                          className="browse-tags__tag-filter-label"
                          title={tag}
                        >
                          {tag}
                        </span>
                        <span className="browse-tags__tag-filter-count">
                          {count}
                        </span>
                      </button>
                    )
                  })}
                </div>
              )}
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      )}

      <div className="search__body">
        {/* 筛选摘要使用即时输入状态，加载和无结果提示依次按请求状态与命中数量决定。 */}
        {hasImmediateFilters && (
          <div className="browse-results__toolbar">
            <ListViewStatus className="search__result-summary">
              <span className="browse-results__summary">
                <span className="browse-results__summary-count">
                  <Trans
                    i18nKey="results.resultSummary"
                    count={totalCount}
                    ns="search"
                    components={{ bold: <strong /> }}
                  />
                </span>
                <span
                  className="browse-results__summary-text"
                  title={activeFilterLabel}
                >
                  {' '}
                  <strong>{activeFilterLabel}</strong>
                </span>
              </span>
            </ListViewStatus>
            {activeTags.length > 0 && (
              <button
                type="button"
                className="browse-results__clear"
                onClick={clearActiveTags}
                title={t('tags.clearFilter')}
                data-testid="search-tags-clear-button"
              >
                <X size={12} />
              </button>
            )}
          </div>
        )}

        {loading && hasImmediateFilters && (
          <ListView
            as="div"
            className="search__results-view"
            contentClassName="search__content"
          >
            <ListViewStatus className="search__result-summary">
              {t('results.loading')}
            </ListViewStatus>
          </ListView>
        )}

        {!loading && hasImmediateFilters && results.length === 0 && (
          <ListView
            as="div"
            className="search__results-view"
            contentClassName="search__content"
          >
            <ListViewStatus className="search__result-summary">
              {t('results.noResults')}
            </ListViewStatus>
          </ListView>
        )}

        {!loading && results.length > 0 && (
          <ListView
            as="div"
            className="search__results-view"
            contentClassName="search__content"
            testId="search-results-list"
            footer={
              <div className="search__pagination">
                <Pagination
                  total={totalCount}
                  page={page}
                  limit={limit}
                  onPageChange={(newPage) => {
                    // 翻页立即使旧请求失效，并把键盘选中位置重置到当前页首项。
                    invalidatePendingRequests()
                    setLoading(true)
                    setPage(newPage)
                    setActiveIndex(0)
                  }}
                />
              </div>
            }
          >
            <ListViewList>
              {results.map((item, index) => (
                <SearchResultCard
                  key={`${item.page_id}-${item.kind}-${index}`}
                  ref={(element) => {
                    resultRefs.current[index] = element
                  }}
                  item={item}
                  isSelected={index === clampedActiveIndex}
                  onMouseEnter={() => setActiveIndex(index)}
                  onFocus={() => setActiveIndex(index)}
                />
              ))}
            </ListViewList>
          </ListView>
        )}
      </div>
    </div>
  )
}
