import { useLayoutEffect } from 'react'
import { useResolvedDesignMode } from '@/features/designtoggle/useResolvedDesignMode'

// The document and editor use the same resolved theme, including system changes.
export default function useApplyDesignMode() {
  const mode = useResolvedDesignMode()
  useLayoutEffect(() => {
    document.documentElement.classList.toggle('dark', mode === 'dark')
  }, [mode])
}
