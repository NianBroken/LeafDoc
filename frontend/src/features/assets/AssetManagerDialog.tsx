// 此对话框将页面附件管理器接到编辑器引用，插入正文后关闭，重命名只通知正文替换。
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/components/ui/dialog'
import { DIALOG_ASSET_MANAGER } from '@/shared/lib/registries/instances'
import { createHotkeyDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { useDialogsStore } from '@/shared/state/dialogs'
import { HotKeyDefinition, useHotKeysStore } from '@/shared/state/hotkeys'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { AssetManager } from '@/features/assets/AssetManager'

// 编辑器引用允许尚未挂载，操作使用可选调用；文件名替换能力也是可选方法。
export type AssetManagerDialogProps = {
  pageId: string
  editorRef: React.RefObject<{
    insertAtCursor: (md: string) => void
    replaceFilenameInMarkdown?: (before: string, after: string) => void
  }>
  onAssetVersionChange: () => void
  isRenamingRef: React.RefObject<boolean>
}

// 打开状态由全局对话框类型决定，不在组件内部另设打开标记。
export function AssetManagerDialog(props: AssetManagerDialogProps) {
  const { t } = useTranslation('assets')
  const { pageId, editorRef, onAssetVersionChange, isRenamingRef } = props
  const closeDialog = useDialogsStore((s) => s.closeDialog)
  const open = useDialogsStore((s) => s.dialogType === DIALOG_ASSET_MANAGER)
  const registerHotkey = useHotKeysStore((s) => s.registerHotkey)
  const unregisterHotkey = useHotKeysStore((s) => s.unregisterHotkey)

  useEffect(() => {
    // 打开时重置共享编辑标记。关闭快捷键始终登记，但仅在本对话框打开时生效。
    if (open) {
      isRenamingRef.current = false
    }

    const cancelHotkey: HotKeyDefinition = createHotkeyDefinition(
      'dialog.close',
      () => {
        if (open) {
          closeDialog()
        }
      },
    )

    registerHotkey(cancelHotkey)
    return () => {
      unregisterHotkey(cancelHotkey)
    }
  }, [open, isRenamingRef, registerHotkey, unregisterHotkey, closeDialog])

  return (
    <Dialog
      open={open}
      onOpenChange={(onOpen) => {
        if (!onOpen) {
          closeDialog()
        }
      }}
    >
      <DialogContent
        className="asset-manager-dialog__content"
        onEscapeKeyDown={(e: KeyboardEvent) => {
          // 禁止原语自行关闭，统一交给全局对话框快捷键分发。
          e.preventDefault()
        }}
      >
        <DialogHeader>
          <DialogTitle>{t('dialog.title')}</DialogTitle>
          <DialogDescription>{t('dialog.description')}</DialogDescription>
        </DialogHeader>
        <AssetManager
          pageId={pageId}
          onAssetVersionChange={onAssetVersionChange}
          onInsert={(md) => {
            // 引用为空时不插入，但仍按同一路径关闭对话框。
            editorRef.current?.insertAtCursor(md)
            closeDialog()
          }}
          onFilenameChange={(before, after) => {
            // 只通知具备替换方法的编辑器，不因重命名主动关闭管理器。
            editorRef.current?.replaceFilenameInMarkdown?.(before, after)
          }}
          isRenamingRef={isRenamingRef}
        />
      </DialogContent>
    </Dialog>
  )
}
