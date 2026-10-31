// 编辑器共享状态管理预览可见性、预览排列、自动换行与自动保存状态。
// 界面偏好持久化，保存中的临时状态只保存在当前内存，不随下次加载恢复。
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// 三种状态只描述供界面显示的当前值，状态转换由自动保存流程调用动作完成。
export type AutoSaveStatus = 'idle' | 'saving' | 'paused'

type EditorStore = {
  previewVisible: boolean
  setPreviewVisible: (visible: boolean) => void
  togglePreview: () => void
  previewStacked: boolean
  togglePreviewLayout: () => void
  lineWrap: boolean
  toggleLineWrap: () => void
  autoSaveStatus: AutoSaveStatus
  setAutoSaveStatus: (status: AutoSaveStatus) => void
}

export const useEditorStore = create<EditorStore>()(
  persist(
    // 默认显示并排预览、开启换行且没有进行中的保存；反转动作读取调用时的最新值。
    (set, get) => ({
      previewVisible: true,
      setPreviewVisible: (visible) => set({ previewVisible: visible }),
      togglePreview: () => set({ previewVisible: !get().previewVisible }),
      previewStacked: false,
      togglePreviewLayout: () => set({ previewStacked: !get().previewStacked }),
      lineWrap: true,
      toggleLineWrap: () => set({ lineWrap: !get().lineWrap }),
      autoSaveStatus: 'idle',
      setAutoSaveStatus: (status) => set({ autoSaveStatus: status }),
    }),
    {
      name: 'leafdoc-editor-settings',
      partialize: (state) => ({
        // 只序列化这三个布尔偏好，排除动作函数与 autoSaveStatus。
        previewVisible: state.previewVisible,
        previewStacked: state.previewStacked,
        lineWrap: state.lineWrap,
      }),
    },
  ),
)
