// 基础对话框通过全局类型决定打开状态，统一标题、说明、按钮和对话框快捷键。

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/components/ui/dialog'
import { createHotkeyDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { useDialogsStore } from '@/shared/state/dialogs'
import { HotKeyDefinition, useHotKeysStore } from '@/shared/state/hotkeys'
import { Loader2 } from 'lucide-react'
import { ReactNode, useEffect } from 'react'
import { Button } from '@/shared/components/ui/button'

// 关闭回调返回值仅在取消按钮点击路径中决定是否关闭，其他关闭路径调用后直接关闭。
// 确认回调返回 true 才关闭；调用方负责 loading、disabled 和异常处理。
export type BaseDialogProps = {
  dialogTitle: string
  dialogDescription: string
  dialogType: string
  defaultAction?: 'confirm' | 'cancel'
  testidPrefix?: string
  onClose: () => boolean
  onConfirm: (type: string) => Promise<boolean>
  children?: ReactNode
  cancelButton: BaseDialogCancelButton
  buttons?: BaseDialogConfirmButton[]
  contentClassName?: string
}

// 取消按钮的禁用只影响该按钮，不同时禁用 Escape 或遮罩触发的关闭路径。
export type BaseDialogCancelButton = {
  label?: string
  variant?:
    'default' | 'destructive' | 'outline' | 'ghost' | 'link' | 'secondary'
  disabled?: boolean
  autoFocus?: boolean
}

// 确认按钮通过 actionType 区分动作，loading 同时显示图标并禁用点击。
export type BaseDialogConfirmButton = {
  label: string
  variant?:
    'default' | 'destructive' | 'outline' | 'ghost' | 'link' | 'secondary'
  loading?: boolean
  disabled?: boolean
  autoFocus?: boolean
  /**
   * 点击按钮时原样传给 onConfirm 的动作名，不限制为固定枚举。
   * 键盘确认路径固定传 confirm，不自动选用某个按钮的 actionType。
   */
  actionType: string
}

// BaseDialog 的 open 来自全局存储，关闭后组件保留多久由外层 DialogManager 决定。
export default function BaseDialog({
  dialogTitle,
  dialogDescription,
  dialogType,
  onClose,
  onConfirm,
  defaultAction = 'confirm',
  children,
  testidPrefix,
  cancelButton,
  buttons,
  contentClassName,
}: BaseDialogProps) {
  const closeDialog = useDialogsStore((s) => s.closeDialog)
  const open = useDialogsStore((s) => s.dialogType === dialogType)
  const registerHotkey = useHotKeysStore((s) => s.registerHotkey)
  const unregisterHotkey = useHotKeysStore((s) => s.unregisterHotkey)

  useEffect(() => {
    // 仅在当前类型打开时登记两个定义，依赖变化或关闭时按对象引用注销。
    if (!open) {
      return
    }

    const confirmHotkey: HotKeyDefinition = createHotkeyDefinition(
      'dialog.confirm',
      async () => {
        // 默认动作设为取消时直接调用关闭回调并清空对话框，不判断返回值。
        if (defaultAction === 'cancel') {
          onClose()
          closeDialog()
          return
        }

        // 至少一个确认按钮既未禁用也未加载，键盘才允许执行固定 confirm 动作。
        if (!buttons || buttons.length === 0) {
          return
        }
        const hasEnabledButton = buttons.some(
          (button) => !button.disabled && !button.loading,
        )
        if (!hasEnabledButton) {
          return
        }

        const result = await onConfirm('confirm')
        if (result) {
          closeDialog()
        }
      },
    )
    // 取消快捷键不读取 cancelButton.disabled，也不使用 onClose 的返回值。
    const cancelHotkey: HotKeyDefinition = createHotkeyDefinition(
      'dialog.close',
      () => {
        onClose()
        closeDialog()
      },
    )
    registerHotkey(confirmHotkey)
    registerHotkey(cancelHotkey)

    return () => {
      unregisterHotkey(confirmHotkey)
      unregisterHotkey(cancelHotkey)
    }
  }, [
    open,
    onClose,
    onConfirm,
    closeDialog,
    dialogType,
    registerHotkey,
    unregisterHotkey,
    defaultAction,
    buttons,
  ])

  return (
    <Dialog
      open={open}
      // 原语请求关闭时调用回调后直接清空全局类型。
      onOpenChange={(isOpen) => {
        if (!isOpen) {
          onClose()
          closeDialog()
        }
      }}
    >
      <DialogContent
        className={contentClassName}
        onEscapeKeyDown={(e: KeyboardEvent) => {
          // 阻止原语自行处理 Escape，取消动作由全局快捷键处理器分发。
          e.preventDefault()
        }}
      >
        <DialogHeader>
          <DialogTitle>{dialogTitle}</DialogTitle>
          <DialogDescription>{dialogDescription}</DialogDescription>
        </DialogHeader>
        {children}
        <div className="base-dialog__footer">
          {cancelButton && (
            <Button
              autoFocus={cancelButton.autoFocus}
              onClick={() => {
                // 点击取消按钮是唯一使用 onClose 布尔结果决定是否关闭的路径。
                const result = onClose()
                if (result) {
                  closeDialog()
                }
              }}
              disabled={cancelButton.disabled}
              data-testid={
                testidPrefix ? `${testidPrefix}-button-cancel` : undefined
              }
              variant={
                cancelButton.variant as
                  | 'default'
                  | 'destructive'
                  | 'outline'
                  | 'ghost'
                  | 'link'
                  | 'secondary'
              }
            >
              {cancelButton.label}
            </Button>
          )}
          {buttons?.map((button) => (
            <Button
              key={button.actionType}
              onClick={async () => {
                // 使用当前按钮动作名等待确认，false 保留对话框，true 清空全局类型。
                const result = await onConfirm(button.actionType)
                if (result) {
                  closeDialog()
                }
              }}
              disabled={button.loading || button.disabled}
              variant={
                button.variant as
                  | 'default'
                  | 'destructive'
                  | 'outline'
                  | 'ghost'
                  | 'link'
                  | 'secondary'
              }
              autoFocus={button.autoFocus}
              data-testid={
                testidPrefix
                  ? `${testidPrefix}-button-${button.actionType}`
                  : undefined
              }
            >
              {button.loading && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              {button.label}
            </Button>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  )
}
