// 本文件封装重新同步任务的触发和状态读取，不在这里轮询或执行重建。
import { fetchWithAuth } from '@/shared/lib/api/client'

// 两个请求共用管理员重同步入口。
const RESYNC_URL = '/api/admin/resync'

// 状态描述当前任务，phase 允许为空，error 可缺省。
export interface ResyncStatus {
  running: boolean
  phase: string | null
  done: boolean
  error?: string
}

// POST 仅等待触发响应，不把任务启动成功当作任务已完成。
export async function triggerResync(): Promise<void> {
  await fetchWithAuth(RESYNC_URL, {
    method: 'POST',
  })
}

// 每次调用重新发送状态请求，不保留模块级缓存。
export async function getResyncStatus(): Promise<ResyncStatus> {
  return await fetchWithAuth<ResyncStatus>(`${RESYNC_URL}/status`, {})
}
