import ScrollableContainer from '@/shared/components/ScrollableContainer'
import { TooltipWrapper } from '@/shared/components/TooltipWrapper'
import {
  panelItemRegistry,
  SIDEBAR_SEARCH_PANEL_ID,
} from '@/shared/lib/registries/instances'
import { createHotkeyDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useHotKeysStore } from '@/shared/state/hotkeys'
import { useSidebarStore } from '@/features/sidebar/state/sidebar'
import { JSX, Suspense, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

const registeredItems = panelItemRegistry.getAllItems()
const sidebarShortcutIds: Partial<
  Record<string, 'sidebar.explorer.open' | 'sidebar.search.open'>
> = {
  tree: 'sidebar.explorer.open',
  search: 'sidebar.search.open',
}

export default function Sidebar() {
  // 面板标签延迟读取翻译，语言改变时重新渲染以同步当前全局语言。
  const { i18n } = useTranslation()
  const appMode = useAppMode()
  const sidebarMode = useSidebarStore((state) => state.sidebarMode)
  const setSidebarMode = useSidebarStore((state) => state.setSidebarMode)
  const setSidebarVisible = useSidebarStore((state) => state.setSidebarVisible)
  const requestSearchFocus = useSidebarStore(
    (state) => state.requestSearchFocus,
  )

  const items = useMemo(
    () =>
      registeredItems.filter((item) => {
        if (item.modes && !item.modes.includes(appMode)) return false
        if (item.isEnabled && !item.isEnabled()) return false
        return true
      }),
    [appMode],
  )

  const tabs: { id: string; label: string; icon: () => JSX.Element }[] =
    useMemo(
      () =>
        items.map((item) => ({
          id: item.id,
          label: item.label(),
          icon: item.icon,
        })),
      // label 从全局国际化实例读取，语言改变时需重新计算映射结果。
      // eslint-disable-next-line react-hooks/exhaustive-deps
      [items, i18n.language],
    )

  useEffect(() => {
    if (items.length === 0) return

    const hasActiveItem = items.some((item) => item.id === sidebarMode)
    if (!hasActiveItem) {
      setSidebarMode(items[0].id)
    }
  }, [items, setSidebarMode, sidebarMode])

  // add hotkeys for each tab
  const registerHotkey = useHotKeysStore((s) => s.registerHotkey)
  const unregisterHotkey = useHotKeysStore((s) => s.unregisterHotkey)

  // 在列表映射之外创建稳定动作引用，避免每次渲染都重建回调。
  const actions = useMemo(() => {
    const actionMap = new Map<string, () => void>()
    items.forEach((item) => {
      actionMap.set(item.id, () => {
        const state = useSidebarStore.getState()
        const alreadyActive =
          state.sidebarVisible && state.sidebarMode === item.id
        setSidebarVisible(true)
        setSidebarMode(item.id)
        if (alreadyActive && item.id === SIDEBAR_SEARCH_PANEL_ID) {
          requestSearchFocus()
        }
      })
    })
    return actionMap
  }, [items, setSidebarMode, setSidebarVisible, requestSearchFocus])

  // 快捷键定义复用稳定动作引用，避免定义对象随渲染变化。
  const hotKeyDefs = useMemo(
    () =>
      items
        .map((item) => {
          const shortcutId = sidebarShortcutIds[item.id]
          if (!shortcutId) return null

          return createHotkeyDefinition(shortcutId, actions.get(item.id)!)
        })
        .filter(Boolean) as {
        keyCombo: string
        enabled: boolean
        action: () => void
        mode: string[]
      }[],
    [items, actions],
  )

  useEffect(() => {
    hotKeyDefs.forEach((hotKeyDef) => {
      registerHotkey(hotKeyDef)
    })
    return () => {
      hotKeyDefs.forEach((hotKeyDef) => {
        unregisterHotkey(hotKeyDef)
      })
    }
  }, [hotKeyDefs, registerHotkey, unregisterHotkey])

  return (
    <aside
      key={'sidebar'}
      data-testid="sidebar"
      id="sidebar"
      className="sidebar"
    >
      {/*
        The actual width is controlled by the parent container (AppLayout)
        so this element just stretches to full width.
      */}
      <div className="sidebar__inner">
        {tabs.length > 0 ? (
          <div className="sidebar__tabs">
            <div className="sidebar__tabs-list">
              {tabs.map((tab) => (
                <TooltipWrapper
                  label={tab.label}
                  key={tab.id}
                  parentClassName="min-w-0"
                >
                  <button
                    data-testid={`sidebar-${tab.id}-tab-button`}
                    onClick={() => setSidebarMode(tab.id)}
                    className={`sidebar__tab-button ${
                      sidebarMode === tab.id
                        ? 'sidebar__tab-button--active'
                        : 'sidebar__tab-button--inactive'
                    }`}
                  >
                    {tab.icon()} <span className="truncate">{tab.label}</span>
                  </button>
                </TooltipWrapper>
              ))}
            </div>
          </div>
        ) : null}
        <div className={`sidebar__content`}>
          {items.map((item) => (
            <ScrollableContainer key={item.id} hidden={sidebarMode !== item.id}>
              <Suspense fallback={null}>
                {item.render({ active: sidebarMode === item.id })}
              </Suspense>
            </ScrollableContainer>
          ))}
        </div>
      </div>
    </aside>
  )
}
