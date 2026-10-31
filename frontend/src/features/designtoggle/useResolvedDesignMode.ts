// 将用户选择与实时系统配色偏好合成为界面使用的浅色或深色值。
import { useSyncExternalStore } from 'react'
import { useDesignModeStore } from './designmode'

const SYSTEM_LIGHT_QUERY = '(prefers-color-scheme: light)'

// 每次订阅监听同一媒体查询，取消订阅时移除对应回调；无浏览器查询能力时返回空清理函数。
function subscribe(onChange: () => void) {
  if (
    typeof window === 'undefined' ||
    typeof window.matchMedia !== 'function'
  ) {
    return () => {}
  }
  const query = window.matchMedia(SYSTEM_LIGHT_QUERY)
  query.addEventListener('change', onChange)
  return () => query.removeEventListener('change', onChange)
}

export function useResolvedDesignMode(): 'light' | 'dark' {
  // 系统偏好变化通过外部状态订阅触发渲染，服务端快照及缺少 matchMedia 时采用浅色。
  const mode = useDesignModeStore((state) => state.mode)
  const prefersLight = useSyncExternalStore(
    subscribe,
    () =>
      typeof window.matchMedia !== 'function' ||
      window.matchMedia(SYSTEM_LIGHT_QUERY).matches,
    () => true,
  )
  // 明确选择浅色或深色时不使用系统结果，system 才根据媒体查询返回颜色。
  return mode === 'system' ? (prefersLight ? 'light' : 'dark') : mode
}
