// 此组件在副作用中调用统一外部跳转函数，本身不绘制页面内容。
import { useEffect } from 'react'
import { redirectToExternal } from '@/shared/lib/redirectToExternal'

// 目标或返回位置变化时重新执行跳转，returnTo 由公共函数编码到 redirect_uri。
export default function ExternalRedirect({
  to,
  returnTo,
}: {
  to: string
  returnTo?: string
}) {
  useEffect(() => {
    redirectToExternal(to, returnTo)
  }, [to, returnTo])
  return null
}
