// 使用任务凭据和顺序状态响应检查恢复状态容器，接口替身不实际上传或恢复数据。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useRestoreStore } from '@/features/restore/state/restore'

vi.mock('@/shared/lib/api/restore', () => ({
  triggerRestore: vi.fn(),
  triggerRestoreFromUpload: vi.fn(),
  getRestoreStatus: vi.fn(),
  triggerSelfRestart: vi.fn(),
}))

import * as restoreAPI from '@/shared/lib/api/restore'

const initialState = useRestoreStore.getState()

describe('useRestoreStore', () => {
  beforeEach(() => {
    // 整体恢复模块创建时的状态，保留真实动作函数，并清除接口调用记录。
    useRestoreStore.setState(initialState, true)
    vi.clearAllMocks()
  })

  it('trigger(id) polls with its job receipt and requires login after complete', async () => {
    // 状态先报告替换进行中再报告完成，检查轮询使用启动响应中的凭据并进入重新登录状态。
    ;(restoreAPI.triggerRestore as ReturnType<typeof vi.fn>).mockResolvedValue({
      statusToken: 'job-token',
    })
    ;(
      restoreAPI.getRestoreStatus as ReturnType<typeof vi.fn>
    ).mockResolvedValueOnce({ running: true, phase: 'swapping', done: false })
    ;(
      restoreAPI.getRestoreStatus as ReturnType<typeof vi.fn>
    ).mockResolvedValueOnce({ running: true, phase: null, done: true })

    await useRestoreStore.getState().trigger('snapshot-1')

    expect(restoreAPI.triggerRestore).toHaveBeenCalledWith('snapshot-1')
    expect(restoreAPI.getRestoreStatus).toHaveBeenCalledWith('job-token')
    const state = useRestoreStore.getState()
    expect(state.isLoading).toBe(false)
    expect(state.needsIntervention).toBe(false)
    expect(state.resyncConfirmed).toBe(true)
    expect(state.requiresLogin).toBe(true)
  })

  it('triggerUpload(file) drives the same phase/resync-tail transitions as trigger(id)', async () => {
    // 上传入口以验证阶段开始，最终断言共享轮询完成后的标记，不断言每个中间渲染状态。
    const file = new File(['zip-bytes'], 'backup.zip')
    ;(
      restoreAPI.triggerRestoreFromUpload as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ statusToken: 'upload-token' })
    ;(
      restoreAPI.getRestoreStatus as ReturnType<typeof vi.fn>
    ).mockResolvedValueOnce({ running: true, phase: 'validating', done: false })
    ;(
      restoreAPI.getRestoreStatus as ReturnType<typeof vi.fn>
    ).mockResolvedValueOnce({ running: true, phase: null, done: true })

    await useRestoreStore.getState().triggerUpload(file)

    expect(restoreAPI.triggerRestoreFromUpload).toHaveBeenCalledWith(file)
    expect(restoreAPI.getRestoreStatus).toHaveBeenCalledWith('upload-token')
    const state = useRestoreStore.getState()
    expect(state.isLoading).toBe(false)
    expect(state.needsIntervention).toBe(false)
    expect(state.resyncConfirmed).toBe(true)
  })

  it('triggerUpload(file) surfaces needsIntervention the same way trigger(id) does', async () => {
    // 完成响应明确要求人工干预，动作正常返回并保留对应提示标记。
    const file = new File(['zip-bytes'], 'backup.zip')
    ;(
      restoreAPI.triggerRestoreFromUpload as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ statusToken: 'upload-token' })
    ;(
      restoreAPI.getRestoreStatus as ReturnType<typeof vi.fn>
    ).mockResolvedValueOnce({
      running: false,
      phase: null,
      done: true,
      needsIntervention: true,
    })

    await useRestoreStore.getState().triggerUpload(file)

    const state = useRestoreStore.getState()
    expect(state.isLoading).toBe(false)
    expect(state.needsIntervention).toBe(true)
  })

  it('triggerUpload(file) rethrows and resets isLoading when the upload call itself fails', async () => {
    // 上传失败时尚无任务凭据，因此断言错误向外传播、加载结束且没有查询状态。
    const file = new File(['zip-bytes'], 'backup.zip')
    const uploadError = new Error('upload failed')
    ;(
      restoreAPI.triggerRestoreFromUpload as ReturnType<typeof vi.fn>
    ).mockRejectedValue(uploadError)

    await expect(
      useRestoreStore.getState().triggerUpload(file),
    ).rejects.toThrow('upload failed')

    expect(useRestoreStore.getState().isLoading).toBe(false)
    expect(restoreAPI.getRestoreStatus).not.toHaveBeenCalled()
  })
})
