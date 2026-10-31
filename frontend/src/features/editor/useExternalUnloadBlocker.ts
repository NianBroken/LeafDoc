// 编辑器通过浏览器离开事件保护未保存内容，页面内路由拦截由其他组件负责。
import { useEffect } from 'react'

// 标志变化时撤销旧监听再绑定新监听，卸载时移除同一个事件处理函数。
export function useExternalUnloadBlocker(shouldBlock: boolean) {
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      // 标志关闭时不干预离开，开启时请求浏览器执行自身的离开确认流程。
      // 提示文案由浏览器决定，空 returnValue 不设置自定义提示文字。
      if (!shouldBlock) return
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [shouldBlock])
}
