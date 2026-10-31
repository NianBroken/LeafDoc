// 将已解析的显示主题同步到根元素，CSS 通过 dark 类控制深色界面。
import { useLayoutEffect } from 'react'
import { useResolvedDesignMode } from '@/features/designtoggle/useResolvedDesignMode'

// useApplyDesignMode 在主题变化时设置或移除 dark 类，主题解析由共享 Hook 完成。
// 布局副作用在绘制前执行，不在卸载时恢复旧类值。
export default function useApplyDesignMode() {
  const mode = useResolvedDesignMode()
  useLayoutEffect(() => {
    document.documentElement.classList.toggle('dark', mode === 'dark')
  }, [mode])
}
