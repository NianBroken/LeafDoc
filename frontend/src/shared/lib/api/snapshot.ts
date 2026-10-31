// 本文件封装完整快照列表、调度设置和下载路径，不在浏览器打包实例文件。
import { API_BASE_URL } from '@/shared/lib/config'
import { fetchWithAuth } from '@/shared/lib/api/client'

// 状态、列表和手动触发入口在本模块集中定义。
const SNAPSHOT_STATUS_URL = '/api/admin/snapshot/status'
const SNAPSHOT_LIST_URL = '/api/admin/snapshot'
const SNAPSHOT_TRIGGER_URL = '/api/admin/snapshot'

// 快照列表项只含标识、生成时间和字节数，内容通过独立下载接口取得。
export interface SnapshotEntry {
  id: string
  createdAt: string
  sizeBytes: number
}

// 功能关闭时详细状态和设置可缺省，消费者不能无条件读取嵌套字段。
export interface SnapshotStatusResponse {
  enabled: boolean
  retentionCount?: number
  settings?: SnapshotSettings
  status?: {
    isRunning: boolean
    lastSnapshotAt: string | null
    lastError: string
    lastPruneError?: string
  }
}

// 两种调度模式共用配置结构，间隔文本、时区与保留数量由服务端解释和校验。
export interface SnapshotSettings {
  enabled: boolean
  runOnStartup: boolean
  mode: 'interval' | 'daily'
  interval: string
  hour: number
  minute: number
  timeZone: string
  retentionCount: number
}

// 单次读取快照状态，不在这里自动轮询。
export async function fetchSnapshotStatus(): Promise<SnapshotStatusResponse> {
  return await fetchWithAuth<SnapshotStatusResponse>(SNAPSHOT_STATUS_URL, {})
}

// 提交完整设置并提取响应 settings，不用输入对象替代服务端确认结果。
export async function updateSnapshotSettings(
  settings: SnapshotSettings,
): Promise<SnapshotSettings> {
  const response = await fetchWithAuth<{ settings: SnapshotSettings }>(
    '/api/admin/snapshot/settings',
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(settings),
    },
  )
  return response.settings
}

// 列表字段为 null 或 undefined 时回退空数组，不对其他错误结构作运行时转换。
export async function fetchSnapshots(): Promise<SnapshotEntry[]> {
  const res = await fetchWithAuth<{ snapshots: SnapshotEntry[] }>(
    SNAPSHOT_LIST_URL,
    {},
  )
  return res.snapshots ?? []
}

// 等待手动触发响应后结束，不代表快照任务已经完成。
export async function triggerSnapshot(): Promise<void> {
  await fetchWithAuth(SNAPSHOT_TRIGGER_URL, {
    method: 'POST',
  })
}

// 编码快照标识后发送删除请求，不在本地先删除列表条目。
export async function deleteSnapshot(id: string): Promise<void> {
  await fetchWithAuth(`/api/admin/snapshot/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

// snapshotDownloadUrl 只构造带基础路径的下载地址，供链接等浏览器原生下载入口使用。
// 不通过 JSON 请求客户端读取 ZIP 字节，也不在此创建下载任务。
export function snapshotDownloadUrl(id: string): string {
  return `${API_BASE_URL}/api/admin/snapshot/${encodeURIComponent(id)}/download`
}
