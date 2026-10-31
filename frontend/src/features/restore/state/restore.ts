// 编排已有快照和上传文件两种恢复入口，取得任务凭据后持续轮询服务端状态。
// 状态用于恢复界面的阶段展示、干预提示和成功后的重新登录引导。
import { create } from 'zustand'
import i18next from '@/shared/lib/i18n'
import {
  triggerRestore,
  triggerRestoreFromUpload,
  getRestoreStatus,
  triggerSelfRestart,
  RestoreStatus,
} from '@/shared/lib/api/restore'
import { sleep } from '@/shared/lib/sleep'

const POLL_INTERVAL_MS = 800
const POLL_ERROR_LIMIT = 3

interface RestoreState {
  isLoading: boolean
  requiresLogin: boolean
  phase: string | null
  // 此标记初始化及触发时均为 false，轮询不将其置为 true，实际阶段由 phase 表达。
  isResyncPhase: boolean
  needsIntervention: boolean
  versionWarning: string | null
  // 此确认标记初始化及触发时为 true，当前状态动作不根据阶段将其置为 false。
  resyncConfirmed: boolean
  trigger: (id: string) => Promise<void>
  triggerUpload: (file: File) => Promise<void>
  selfRestart: () => Promise<void>
}

export const useRestoreStore = create<RestoreState>((set) => ({
  isLoading: false,
  requiresLogin: false,
  phase: null,
  isResyncPhase: false,
  needsIntervention: false,
  versionWarning: null,
  resyncConfirmed: true,

  trigger: async (id: string) => {
    // 重设任务展示状态再提交快照 ID，此处不清空 requiresLogin，也不阻止并行触发。
    set({
      isLoading: true,
      phase: null,
      isResyncPhase: false,
      needsIntervention: false,
      versionWarning: null,
      resyncConfirmed: true,
    })
    try {
      const receipt = await triggerRestore(id)
      await pollUntilDone(set, receipt.statusToken)
    } catch (err) {
      set({ isLoading: false })
      throw err
    }
  },

  triggerUpload: async (file: File) => {
    // 上传完成取得任务凭据后进入同一轮询流程，上传本身失败时直接结束加载并抛出。
    set({
      isLoading: true,
      phase: null,
      isResyncPhase: false,
      needsIntervention: false,
      versionWarning: null,
      resyncConfirmed: true,
    })
    try {
      const receipt = await triggerRestoreFromUpload(file)
      await pollUntilDone(set, receipt.statusToken)
    } catch (err) {
      set({ isLoading: false })
      throw err
    }
  },

  selfRestart: async () => {
    // 发出服务端重启请求，界面不接收该动作的拒绝结果。
    try {
      await triggerSelfRestart()
    } catch {
      // 服务进程替换期间请求连接会中断；此分支统一忽略请求错误，不区分具体错误原因。
    }
  },
}))

// 两种入口使用相同状态凭据轮询，每轮请求前等待固定间隔，不设总轮数或总时长上限。
// 连续请求失败达到门槛才抛出；任一次成功响应都将连续失败计数归零。
async function pollUntilDone(
  set: (partial: Partial<RestoreState>) => void,
  statusToken: string,
): Promise<void> {
  let consecutiveErrors = 0
  for (;;) {
    await sleep(POLL_INTERVAL_MS)

    let status: RestoreStatus
    try {
      status = await getRestoreStatus(statusToken)
      consecutiveErrors = 0
    } catch (err) {
      consecutiveErrors++
      if (consecutiveErrors >= POLL_ERROR_LIMIT) {
        set({ isLoading: false, phase: null })
        throw err
      }
      continue
    }

    set({
      phase: status.phase || null,
      versionWarning: status.versionWarning || null,
    })

    if (status.done) {
      // 完成状态优先检查人工干预，再检查任务错误；干预分支正常返回，交由界面展示处理入口。
      if (status.needsIntervention) {
        set({ isLoading: false, needsIntervention: true })
        return
      }
      if (status.error) {
        set({ isLoading: false, phase: null })
        throw new Error(status.error)
      }
      set({
        isLoading: false,
        phase: null,
        isResyncPhase: false,
        requiresLogin: true,
      })
      return
    }
    if (!status.running) {
      // 既未完成也未运行的响应无法继续追踪任务，结束加载并抛出本地化的任务丢失错误。
      set({ isLoading: false, phase: null })
      throw new Error(i18next.t('errors.jobLost', { ns: 'restore' }))
    }
  }
}
