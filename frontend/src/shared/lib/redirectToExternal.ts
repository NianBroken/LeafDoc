// 统一通过 location.href 离开当前页面，登录和退出跳转可共用此入口。
// 提供 returnTo 时直接拼接当前 origin 形成返回地址，再写入 redirect_uri。
// 本函数不校验目标协议或域名，输入的允许范围由调用方控制。
export function redirectToExternal(url: string, returnTo?: string) {
  if (!returnTo) {
    window.location.href = url
    return
  }
  const absoluteReturnTo = `${window.location.origin}${returnTo}`

  // 可解析的绝对 URL 用标准参数接口覆盖同名参数，自动处理编码与片段位置。
  try {
    const redirectUrl = new URL(url)
    redirectUrl.searchParams.set('redirect_uri', absoluteReturnTo)
    window.location.href = redirectUrl.toString()
  } catch {
    // 无法按绝对 URL 解析时按文本拼接，将返回参数放到片段之前，保留原片段。
    const hashIndex = url.indexOf('#')
    const base = hashIndex === -1 ? url : url.slice(0, hashIndex)
    const hash = hashIndex === -1 ? '' : url.slice(hashIndex)
    const separator = base.includes('?') ? '&' : '?'
    window.location.href = `${base}${separator}redirect_uri=${encodeURIComponent(absoluteReturnTo)}${hash}`
  }
}
