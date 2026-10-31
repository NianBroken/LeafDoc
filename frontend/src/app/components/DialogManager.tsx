// 根据全局对话框类型渲染注册组件，并在关闭后短暂保留组件供退出动画使用。
import { Suspense, useEffect, useState } from 'react'

import { dialogRegistry } from '@/shared/lib/registries/instances'
import { useDialogsStore } from '@/shared/state/dialogs'

// 模块求值时取得注册器结果，组件本身不订阅后续注册变化。
const dialogs = dialogRegistry.getAllDialogs()

// DialogManager 只渲染 renderType 匹配项，懒组件未加载完成时不显示额外占位内容。
export function DialogManager() {
  const dialogType = useDialogsStore((state) => state.dialogType)
  const dialogProps = useDialogsStore((state) => state.dialogProps)

  // 渲染状态与全局打开状态分开，关闭时先让组件接收到关闭状态，再延迟移除。
  const [renderType, setRenderType] = useState(dialogType)
  const [renderProps, setRenderProps] = useState(dialogProps)

  useEffect(() => {
    // 打开类型或属性变化后，在副作用中同步最近渲染对象。
    if (dialogType !== null) {
      setRenderType(dialogType)
      setRenderProps(dialogProps)
      return
    }
    // 关闭后等待二百毫秒清空渲染对象，依赖变化或卸载会取消尚未触发的定时器。
    const timeoutId = setTimeout(() => {
      setRenderType(null)
      setRenderProps(null)
    }, 200)
    return () => clearTimeout(timeoutId)
  }, [dialogType, dialogProps])

  return (
    <Suspense fallback={null}>
      {dialogs.map((dialog) => {
        if (dialog.type !== renderType) return null
        return dialog.render({ ...renderProps })
      })}
    </Suspense>
  )
}
