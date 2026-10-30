import { useSyncExternalStore } from 'react'
import { useDesignModeStore } from './designmode'

const SYSTEM_LIGHT_QUERY = '(prefers-color-scheme: light)'

// All themed surfaces follow the same live system preference and release listeners.
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
  const mode = useDesignModeStore((state) => state.mode)
  const prefersLight = useSyncExternalStore(
    subscribe,
    () =>
      typeof window.matchMedia !== 'function' ||
      window.matchMedia(SYSTEM_LIGHT_QUERY).matches,
    () => true,
  )
  return mode === 'system' ? (prefersLight ? 'light' : 'dark') : mode
}
