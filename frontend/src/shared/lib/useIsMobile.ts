// 按视口宽度媒体查询判断窄屏，不检测设备型号、触摸能力或浏览器标识。
import { useLayoutEffect, useState } from 'react'

// 宽度不超过 767 CSS 像素时返回 true，无 window 环境初始返回 false。
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia('(max-width: 767px)').matches
  })

  useLayoutEffect(() => {
    if (typeof window === 'undefined') return

    const media = window.matchMedia('(max-width: 767px)')
    const handleChange = () => setIsMobile(media.matches)

    // 仅注册一次查询变化监听，卸载时移除同一个处理函数。
    media.addEventListener('change', handleChange)
    return () => media.removeEventListener('change', handleChange)
  }, [])

  return isMobile
}
