import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/shared/components/ui/dialog'
import { DIALOG_IMAGE_PREVIEW } from '@/shared/lib/registries/instances'
import { createHotkeyDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { useDialogsStore } from '@/shared/state/dialogs'
import { HotKeyDefinition, useHotKeysStore } from '@/shared/state/hotkeys'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

// 对话框注册的关闭快捷键与 Escape 行为共同保证预览可以退出。

type Props = {
  src: string
  alt?: string
}

export function ImagePreviewDialog({ src, alt }: Props) {
  const { t } = useTranslation('viewer')
  const closeDialog = useDialogsStore((s) => s.closeDialog)
  const registerHotkey = useHotKeysStore((s) => s.registerHotkey)
  const unregisterHotkey = useHotKeysStore((s) => s.unregisterHotkey)
  const open = useDialogsStore((s) => s.dialogType === DIALOG_IMAGE_PREVIEW)
  const [natural, setNatural] = useState<{ w: number; h: number } | null>(null)

  useEffect(() => {
    if (!open) {
      return
    }
    const cancelHotkey: HotKeyDefinition = createHotkeyDefinition(
      'dialog.close',
      () => {
        closeDialog()
      },
    )

    setNatural(null)
    registerHotkey(cancelHotkey)

    return () => {
      unregisterHotkey(cancelHotkey)
    }
  }, [open, registerHotkey, unregisterHotkey, closeDialog])

  const dialogWidth = useMemo(() => {
    if (!natural) return 'min(95vw, 900px)'

    const paddingAndChrome = 12 * 2 + 24 // px-6 left/right + slack
    const target = natural.w + paddingAndChrome

    const minW = 360
    const maxW = 1400

    const px = Math.max(minW, Math.min(target, maxW))
    return `min(95vw, ${px}px)`
  }, [natural])

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && closeDialog()}>
      <DialogContent
        className="p-0"
        style={{ width: dialogWidth, maxWidth: '95vw' }}
        onEscapeKeyDown={(e: KeyboardEvent) => {
          e.preventDefault()
        }}
      >
        <DialogHeader className="px-6 pt-6">
          <DialogTitle className="mt-3 flex flex-wrap items-baseline gap-2 leading-[1.2]">
            <span className="truncate">{alt ?? ''}</span>
            {natural ? (
              <span className="text-muted-foreground text-sm font-normal">
                ({natural.w}×{natural.h}px)
              </span>
            ) : null}

            <a
              href={src}
              target="_blank"
              rel="noreferrer noopener"
              className="text-muted-foreground hover:text-foreground ml-auto text-sm underline"
              onClick={(e) => e.stopPropagation()}
            >
              {t('imagePreview.openInNewTab')}
            </a>
          </DialogTitle>
        </DialogHeader>

        <div className="image-preview-frame">
          <img
            src={src}
            alt={alt}
            onLoad={(e) => {
              const img = e.currentTarget
              setNatural({ w: img.naturalWidth, h: img.naturalHeight })
            }}
            className="preview-image bg-muted-surface rounded-lg"
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
