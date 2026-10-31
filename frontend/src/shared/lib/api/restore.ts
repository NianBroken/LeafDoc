// 本文件封装管理员恢复操作和独立进度查询，不在客户端替换实例数据。
import { fetchWithAuth } from '@/shared/lib/api/client'
import { API_BASE_URL } from '@/shared/lib/config'

// 需要管理员认证的恢复操作共用此基础路径。
const RESTORE_URL = '/api/admin/restore'

// 进度由服务端维护，阶段、完成状态、错误和人工处理标记分别传递。
export interface RestoreStatus {
  running: boolean
  phase: string | null
  done: boolean
  error?: string
  versionWarning?: string
  needsIntervention?: boolean
}

// 恢复回执携带专用进度令牌，与浏览器原有账号会话分开使用。
export interface RestoreReceipt {
  statusToken: string
}

// 快照 ID 编码为单个路径片段，POST 成功返回回执，不在这里等待恢复结束。
export async function triggerRestore(id: string): Promise<RestoreReceipt> {
  return await fetchWithAuth<RestoreReceipt>(
    `${RESTORE_URL}/${encodeURIComponent(id)}`,
    {
      method: 'POST',
    },
  )
}

// 将上传文件交给服务端恢复入口，输入校验和归档解析由服务端执行。
export async function triggerRestoreFromUpload(
  file: File,
): Promise<RestoreReceipt> {
  const formData = new FormData()
  formData.append('file', file)
  return await fetchWithAuth<RestoreReceipt>(`${RESTORE_URL}/upload`, {
    method: 'POST',
    body: formData,
    headers: {}, // 表单内容类型及分隔标记由浏览器生成。
  })
}

// getRestoreStatus 仅使用专用头中的回执令牌，明确不携带 Cookie，也不触发会话刷新。
// HTTP 非成功状态抛普通错误，成功 JSON 不作额外结构校验。
export async function getRestoreStatus(
  statusToken: string,
): Promise<RestoreStatus> {
  const response = await fetch(`${API_BASE_URL}/api/restore/progress`, {
    headers: { 'X-Restore-Token': statusToken },
    credentials: 'omit',
  })
  if (!response.ok) throw new Error(`Restore progress: HTTP ${response.status}`)
  return await response.json()
}

// triggerSelfRestart 提交自重启请求，等待统一客户端完成或抛错。
// 本函数不吞掉网络、权限或服务端错误，也不自行等待服务重新可用，后续判断由调用方负责。
export async function triggerSelfRestart(): Promise<void> {
  await fetchWithAuth(`${RESTORE_URL}/self-restart`, {
    method: 'POST',
  })
}
