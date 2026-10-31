// 本文件封装实例公开读取与目录显示设置，不在请求前更改本地配置状态。
import { fetchWithAuth } from '@/shared/lib/api/client'

// 两种设置使用各自的管理员子资源路径。
const PUBLIC_ACCESS_URL = '/api/admin/settings/public-access'
const TOC_DISPLAY_URL = '/api/admin/settings/toc-display'

// 写入响应返回服务端确认的布尔值，不由调用方输入直接代替响应。
export type PublicAccessResponse = {
  enabled: boolean
}

/**
 * 提交运行时公开读取开关，具体权限和固定来源限制由服务端处理。
 * 非成功响应经统一客户端解释，本函数不自行判定匿名请求能访问哪些接口。
 */
export async function setPublicAccess(
  enabled: boolean,
): Promise<PublicAccessResponse> {
  return await fetchWithAuth<PublicAccessResponse>(PUBLIC_ACCESS_URL, {
    method: 'PUT',
    body: JSON.stringify({ enabled }),
  })
}

// 此兼容接口只声明 alwaysShow，不包含完整目录模式字段。
export type TocDisplayResponse = {
  alwaysShow: boolean
}

/**
 * 提交目录显示兼容布尔值，不在此计算正文标题数量或立即切换当前页面目录。
 */
export async function setAlwaysShowToc(
  alwaysShow: boolean,
): Promise<TocDisplayResponse> {
  return await fetchWithAuth<TocDisplayResponse>(TOC_DISPLAY_URL, {
    method: 'PUT',
    body: JSON.stringify({ alwaysShow }),
  })
}
