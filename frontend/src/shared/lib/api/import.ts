// 本文件封装 ZIP 导入计划、执行和取消请求，不在客户端实际解压或修改页面。
import type { PageKind } from '@/shared/contracts/page'
import i18next from '@/shared/lib/i18n'
import { fetchWithAuth } from '@/shared/lib/api/client'

// 计划同时承载待处理项、执行阶段、进度和可选执行结果，字段名保持服务端原样。
export type ImportPlan = {
  id: string
  tree_hash: string
  items: ImportPlanItem[]
  errors: string[]
  execution_status: ImportExecutionStatus
  cancel_requested: boolean
  execution_result?: ImportResult
  execution_error?: string
  processed_items: number
  total_items: number
  current_item_source_path?: string
  started_at?: string
  finished_at?: string
}

// 阶段文本用于显示服务端进度，本类型不提供合法状态转换判断。
export type ImportExecutionStatus =
  'planned' | 'running' | 'completed' | 'failed' | 'canceled'

// 单项计划说明来源与目标路径、冲突、预定动作及目标页面是否已经存在。
export type ImportPlanItem = {
  source_path: string
  target_path: string
  title: string
  desired_slug: string
  kind: PageKind
  exists: boolean
  existing_id: string | null
  action: 'create' | 'update' | 'skip'
  conflicts: string[] | null
  notes: string[] | null
}

// 执行结果区分汇总计数和每项实际动作，并保留执行前后树摘要。
export type ImportResult = {
  imported_count: number
  updated_count: number
  skipped_count: number
  items: {
    source_path: string
    target_path: string
    action: 'created' | 'updated' | 'skipped' | 'conflicted'
    error?: string
  }[]
  tree_hash: string
  tree_hash_before: string
}

// 上传 ZIP 的原始 File 到计划接口，计划生成与文件内容验证在服务端执行。
export async function createImportPlanFromZip(file: File): Promise<ImportPlan> {
  const formData = new FormData()
  formData.append('file', file)

  return await fetchWithAuth<ImportPlan>('/api/import/plan', {
    method: 'POST',
    body: formData,
    headers: {}, // 表单分隔标记由浏览器生成，统一客户端保留 FormData 传输。
  })
}

// 查询当前计划，允许服务端通过 null 表示无计划，不在此轮询。
export async function getImportPlan(): Promise<ImportPlan | null> {
  return await fetchWithAuth<ImportPlan | null>('/api/import/plan', {
    method: 'GET',
  })
}

// 请求执行当前计划，不附加计划 ID 或页面内容，返回的阶段由服务端决定。
export async function executeImportPlan(): Promise<ImportPlan> {
  return await fetchWithAuth<ImportPlan>('/api/import/execute', {
    method: 'POST',
  })
}

// DELETE 请求后只接受 null 或带 execution_status 属性的对象，其他形状抛出提示错误。
// 对象分支仅检查属性存在，不检查其类型、值或计划的其他必需字段。
export async function cancelImportPlan(): Promise<ImportPlan | null> {
  const response = await fetchWithAuth('/api/import/plan', {
    method: 'DELETE',
  })

  if (response === null) {
    return null
  }

  if (
    typeof response === 'object' &&
    response !== null &&
    'execution_status' in response
  ) {
    return response as ImportPlan
  }

  throw new Error(
    i18next.t('toast.unexpectedCancelResponse', { ns: 'importer' }),
  )
}
