// 此组件展示单个页面附件，提供重命名、删除和向编辑器插入引用的操作。
// 浏览器预览地址带部署前缀，写入 Markdown 的引用保留列表返回的原始路径。
import { Button } from '@/shared/components/ui/button'
import { deleteAsset, renameAsset } from '@/shared/lib/api/assets'
import { mapApiError } from '@/shared/lib/api/errors'
import {
  AUDIO_EXTENSIONS,
  IMAGE_EXTENSIONS,
  PDF_EXTENSIONS,
  VIDEO_EXTENSIONS,
} from '@/shared/lib/config'
import { withBasePath } from '@/shared/lib/routePath'
import { createHotkeyDefinition } from '@/shared/lib/shortcuts/shortcutCatalog'
import { HotKeyDefinition, useHotKeysStore } from '@/shared/state/hotkeys'
import { Check, FileText, Link2, Pencil, Play, Trash2, X } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { AssetPreviewTooltip } from '@/features/assets/AssetPreviewTooltip'

// 类型识别复用公共扩展名清单，不读取文件内容或响应 MIME 类型。
const imageExtensions = IMAGE_EXTENSIONS
const audioExtensions = AUDIO_EXTENSIONS
const videoExtensions = VIDEO_EXTENSIONS
const pdfExtensions = PDF_EXTENSIONS

// 编辑中的文件名由父组件共享，保证列表使用同一个编辑目标。
// 内容插入、文件名替换和资源版本变化通过回调传回编辑器。
type Props = {
  pageId: string
  filename: string
  editingFilename: string | null
  setEditingFilename: (filename: string | null) => void
  onAssetVersionChange?: () => void
  onReload: () => void
  onInsert: (md: string) => void
  onFilenameChange?: (before: string, after: string) => void
}

// AssetItem 仅编辑文件末段的基础名称，扩展名从原始路径取得并转为小写。
export function AssetItem({
  pageId,
  filename,
  editingFilename,
  setEditingFilename,
  onReload,
  onInsert,
  onFilenameChange,
  onAssetVersionChange,
}: Props) {
  const { t } = useTranslation('assets')
  const markdownAssetUrl = filename
  // 接受三种资源前缀形式，统一成站内绝对路径后应用部署 basePath。
  const assetPath = filename.startsWith('/assets/')
    ? filename
    : filename.startsWith('assets/')
      ? `/${filename}`
      : `/assets/${filename}`
  const assetUrl = withBasePath(assetPath)
  const ext = filename.split('.').pop()?.toLowerCase()
  const isImage = imageExtensions.includes(ext ?? '')
  const isAudio = audioExtensions.includes(ext ?? '')
  const isVideo = videoExtensions.includes(ext ?? '')
  const isPdf = pdfExtensions.includes(ext ?? '')
  const isPlayableMedia = isAudio || isVideo
  const isEmbeddable = isImage || isPdf
  const baseName = filename.split('/').pop() ?? filename
  const isEditing = editingFilename === filename
  const registerHotkey = useHotKeysStore((s) => s.registerHotkey)
  const unregisterHotkey = useHotKeysStore((s) => s.unregisterHotkey)

  const [newName, setNewName] = useState(baseName.replace(/\.[^/.]+$/, ''))

  const handleRename = useCallback(async () => {
    // 名称拼回原扩展名，相同名称只退出编辑，不发送写请求。
    try {
      const newFilename = `${newName}.${ext}`
      if (newFilename === baseName) {
        setEditingFilename(null)
        return
      }

      await renameAsset(pageId, baseName, newFilename)
      // 请求成功后按顺序通知正文替换、版本变化与列表重载，不在这里直接修改正文。
      toast.success(t('item.renameSuccess'))
      onFilenameChange?.(baseName, newFilename)
      onAssetVersionChange?.()
      onReload()
    } catch (err: unknown) {
      toast.error(mapApiError(err, t('item.renameErrorFallback')).message)
    }
  }, [
    pageId,
    baseName,
    newName,
    ext,
    onReload,
    onFilenameChange,
    onAssetVersionChange,
    setEditingFilename,
    t,
  ])

  const handleDelete = async () => {
    // 删除成功才刷新列表与版本；失败映射提示并保留原列表供继续操作。
    try {
      await deleteAsset(pageId, baseName)
      toast.success(t('item.deleteSuccess'))
      onReload()
      onAssetVersionChange?.()
    } catch (err) {
      toast.error(mapApiError(err, t('item.deleteErrorFallback')).message)
      console.error('Delete failed', err)
    }
  }

  const handleInsertMarkdown = () => {
    // 编辑文件名期间禁止双击插入，避免重命名交互同时改变正文。
    if (isEditing) return
    onInsert(buildInsertMarkdown('default'))
  }

  const handleInsertLink = () => {
    // 显式链接模式对图片和 PDF 也生成普通链接，不使用嵌入语法。
    if (isEditing) return
    onInsert(buildInsertMarkdown('link'))
  }

  const handleInsertPlayer = () => {
    // 播放器只对公共清单识别的音频或视频开放。
    if (isEditing || !isPlayableMedia) return
    onInsert(buildInsertMarkdown('player'))
  }

  const buildInsertMarkdown = (mode: 'default' | 'link' | 'player') => {
    // 音视频模式生成 HTML 控件；默认图片与 PDF 使用图片式嵌入，其他类型用链接。
    // 所有片段末尾带换行，此处不转义文件名中的 Markdown 或 HTML 特殊字符。
    if (mode === 'player' && isPlayableMedia) {
      if (isAudio) {
        return `<audio controls src="${markdownAssetUrl}"></audio>\n`
      }
      return `<video controls preload="metadata" src="${markdownAssetUrl}"></video>\n`
    }

    if (mode === 'link') {
      return `[${baseName}](${markdownAssetUrl})\n`
    }

    return isEmbeddable
      ? `![${baseName}](${markdownAssetUrl})\n`
      : `[${baseName}](${markdownAssetUrl})\n`
  }

  // 只有当前附件处于重命名状态才登记确认与取消快捷键，依赖变化时注销旧定义。
  useEffect(() => {
    if (!isEditing) return

    const enterHotkey: HotKeyDefinition = createHotkeyDefinition(
      'asset.rename.confirm',
      handleRename,
    )

    const escapeHotkey: HotKeyDefinition = createHotkeyDefinition(
      'asset.rename.cancel',
      () => {
        setEditingFilename(null)
        setNewName(baseName.replace(/\.[^/.]+$/, ''))
      },
    )
    registerHotkey(enterHotkey)
    registerHotkey(escapeHotkey)

    return () => {
      unregisterHotkey(enterHotkey)
      unregisterHotkey(escapeHotkey)
    }
  }, [
    isEditing,
    baseName,
    registerHotkey,
    unregisterHotkey,
    handleRename,
    setEditingFilename,
  ])

  return (
    <li
      className="group asset-item"
      onDoubleClick={handleInsertMarkdown}
      data-testid="asset-item"
    >
      <div className="flex min-w-0 flex-1 items-center gap-1">
        {isImage ? (
          <AssetPreviewTooltip url={assetUrl} name={baseName}>
            <img
              src={assetUrl}
              alt={baseName}
              className="asset-item__preview-image"
            />
          </AssetPreviewTooltip>
        ) : (
          <AssetPreviewTooltip url={assetUrl} name={baseName}>
            <div className="asset-item__preview-file">
              <FileText size={18} />
            </div>
          </AssetPreviewTooltip>
        )}

        {isEditing ? (
          // 输入只保存基础名称，取消时从原文件名恢复该输入值。
          <input
            autoFocus
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            className="asset-item__filename--editing-input"
          />
        ) : (
          <span className="asset-item__filename">{baseName}</span>
        )}
      </div>

      <div className="asset-item__actions">
        {/* 编辑状态只显示保存和取消，普通状态按类型显示插入、重命名和删除操作。 */}
        {isEditing ? (
          <>
            <Button
              variant="outline"
              size="icon"
              className="asset-item__action-button asset-item__action-button--save"
              onClick={handleRename}
              title={t('item.save')}
            >
              <Check size={16} />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="asset-item__action-button asset-item__action-button--cancel"
              onClick={() => {
                setEditingFilename(null)
                setNewName(baseName.replace(/\.[^/.]+$/, ''))
              }}
              title={t('item.cancel')}
            >
              <X size={16} />
            </Button>
          </>
        ) : (
          <>
            {isEmbeddable && (
              <Button
                variant="outline"
                size="icon"
                className="asset-item__action-button"
                onClick={(e) => {
                  e.stopPropagation()
                  handleInsertLink()
                }}
                title={
                  isImage ? t('item.insertImageLink') : t('item.insertPdfLink')
                }
                data-testid="asset-insert-link-button"
              >
                <Link2 size={16} />
              </Button>
            )}
            {isPlayableMedia && (
              <Button
                variant="outline"
                size="icon"
                className="asset-item__action-button"
                onClick={(e) => {
                  e.stopPropagation()
                  handleInsertPlayer()
                }}
                title={
                  isAudio
                    ? t('item.insertAudioPlayer')
                    : t('item.insertVideoPlayer')
                }
                data-testid="asset-insert-player-button"
              >
                <Play size={16} />
              </Button>
            )}
            <Button
              variant="outline"
              size="icon"
              className="asset-item__action-button"
              onClick={(e) => {
                e.stopPropagation()
                handleInsertMarkdown()
              }}
              title={
                isImage
                  ? t('item.insertImage')
                  : isPdf
                    ? t('item.insertPdf')
                    : t('item.insertLink')
              }
              data-testid="asset-insert-default-button"
            >
              <FileText size={16} />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="asset-item__action-button"
              onClick={(e) => {
                e.stopPropagation()
                setNewName(baseName.replace(/\.[^/.]+$/, ''))
                setEditingFilename(filename)
              }}
              title={t('item.rename')}
            >
              <Pencil size={16} />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="asset-item__action-button asset-item__action-button--delete"
              onClick={(e) => {
                e.stopPropagation()
                handleDelete()
              }}
              title={t('item.delete')}
            >
              <Trash2 size={16} />
            </Button>
          </>
        )}
      </div>
    </li>
  )
}
