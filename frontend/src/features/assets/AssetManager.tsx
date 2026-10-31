// 此组件按页面标识加载附件列表，支持文件选择与拖放并发上传。
// 文件项操作复用 AssetItem，编辑器交互通过父组件回调连接。
import { getAssets, uploadAsset } from '@/shared/lib/api/assets'
import { mapApiError } from '@/shared/lib/api/errors'
import { formatBytes } from '@/shared/lib/config'
import { useConfigStore } from '@/shared/state/config'
import { UploadCloud } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { AssetItem } from '@/features/assets/AssetItem'

// isRenamingRef 向对话框外共享编辑状态，其他回调传递正文与资源缓存变化。
type Props = {
  pageId: string
  onInsert?: (md: string) => void
  onFilenameChange?: (before: string, after: string) => void
  onAssetVersionChange?: () => void
  isRenamingRef: React.RefObject<boolean>
}

// AssetManager 本地保存列表、拖放样式和上传名称集合，大小上限来自运行配置存储。
export function AssetManager({
  pageId,
  onInsert,
  onFilenameChange,
  onAssetVersionChange,
  isRenamingRef,
}: Props) {
  const { t } = useTranslation('assets')
  const [assets, setAssets] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const fileInput = useRef<HTMLInputElement>(null)
  const dropRef = useRef<HTMLDivElement>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [isHovered, setIsHovered] = useState(false)
  const [editingFilename, setEditingFilename] = useState<string | null>(null)
  const [uploadingFiles, setUploadingFiles] = useState<Set<string>>(new Set())
  const maxAssetUploadSizeBytes = useConfigStore(
    (s) => s.maxAssetUploadSizeBytes,
  )

  const handleSetEditingFilename = (filename: string | null) => {
    // ref 与 React 状态同步写入，调用方无需等待下一次渲染即可读取编辑标记。
    isRenamingRef.current = !!filename
    setEditingFilename(filename)
  }

  const loadAssets = useCallback(
    // 首次读取显示整体加载状态，写操作后的刷新不遮住现有列表。
    // 错误只记录到控制台，函数不向上传调用方继续抛出读取失败。
    async (showLoading = false) => {
      if (showLoading) setLoading(true)
      try {
        const result = await getAssets(pageId)
        setAssets(result)
      } catch (err) {
        console.error('Failed to load assets', err)
      } finally {
        if (showLoading) setLoading(false)
      }
    },
    [pageId],
  )

  useEffect(() => {
    // 页面标识变化重新读取，未安装请求取消或旧结果过滤。
    loadAssets(true)
  }, [pageId, loadAssets])

  const handleUploadFile = async (file: File) => {
    // 客户端按单个 File 的字节数预检，等于上限允许继续，服务器另行限制请求体。
    if (file.size > maxAssetUploadSizeBytes) {
      toast.error(
        t('manager.fileTooLarge', {
          maxSize: formatBytes(maxAssetUploadSizeBytes),
        }),
      )
      return
    }

    setUploadingFiles((prev) => new Set(prev).add(file.name))
    // 集合按名称而非文件对象计数，同名并发文件共用一个展示项。

    try {
      await uploadAsset(pageId, file)
      await loadAssets(false)
      onAssetVersionChange?.()
    } catch (err) {
      console.error('Upload failed', err)
      toast.error(mapApiError(err, t('manager.uploadErrorFallback')).message)
    } finally {
      // 请求路径结束后移除上传名称，使用新 Set 触发状态更新。
      setUploadingFiles((prev) => {
        const next = new Set(prev)
        next.delete(file.name)
        return next
      })
    }
  }

  const handleUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    // 同次选择的文件并行上传，全部结束后清空控件，允许再次选择相同文件。
    const files = Array.from(event.target.files ?? [])
    await Promise.all(files.map(handleUploadFile))
    if (fileInput.current) fileInput.current.value = ''
  }

  const handleDrop = async (e: React.DragEvent<HTMLDivElement>) => {
    // 阻止浏览器打开拖入文件，取消拖入高亮后复用相同上传流程。
    e.preventDefault()
    setIsDragging(false)
    await Promise.all(
      Array.from(e.dataTransfer.files ?? []).map(handleUploadFile),
    )
  }

  const dropzoneClassName = [
    // 拖入态优先于普通悬停态，过滤空项后组成最终样式。
    'asset-manager__dropzone',
    isDragging
      ? 'asset-manager__dropzone--dragging'
      : isHovered
        ? 'asset-manager__dropzone--hover'
        : '',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <div className="asset-manager">
      <div className="asset-manager__title">{t('manager.title')}</div>

      <div
        data-testid="asset-upload-dropzone"
        ref={dropRef}
        onDragOver={(e) => {
          e.preventDefault()
          setIsDragging(true)
        }}
        onDragEnter={() => setIsDragging(true)}
        onDragLeave={() => {
          setIsDragging(false)
          setIsHovered(false)
        }}
        onDrop={handleDrop}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
        onClick={() => fileInput.current?.click()}
        className={dropzoneClassName}
      >
        <UploadCloud className="asset-manager__dropzone-icon" size={20} />
        <p className="asset-manager__dropzone-text">
          {t('manager.dropzoneText')}
        </p>
        <p className="asset-manager__dropzone-text">
          {t('manager.maxFileSize', {
            maxSize: formatBytes(maxAssetUploadSizeBytes),
          })}
        </p>
        <input
          type="file"
          ref={fileInput}
          onChange={handleUpload}
          className="hidden"
          multiple
        />
        {uploadingFiles.size > 0 && (
          <div className="asset-manager__dropzone-uploading">
            {t('manager.uploading', { count: uploadingFiles.size })}
          </div>
        )}
      </div>

      <div className="asset-manager__list-container">
        {loading ? (
          <p className="asset-manager__loading">{t('manager.loading')}</p>
        ) : assets.length === 0 ? (
          <p className="asset-manager__empty">{t('manager.empty')}</p>
        ) : (
          <ul className="asset-manager__list custom-scrollbar">
            {assets.map((filename) => (
              <AssetItem
                key={filename}
                filename={filename}
                editingFilename={editingFilename}
                setEditingFilename={handleSetEditingFilename}
                pageId={pageId}
                onReload={loadAssets}
                onAssetVersionChange={onAssetVersionChange}
                onInsert={(md) => onInsert?.(md)}
                onFilenameChange={onFilenameChange}
              />
            ))}
          </ul>
        )}
      </div>

      <p className="asset-manager__tip">{t('manager.tip')}</p>
    </div>
  )
}
