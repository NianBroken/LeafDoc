// 展示树状态中的固定节点，并通过服务端版本号提交取消固定操作。
import { mapApiError } from '@/shared/lib/api/errors'
import { pinPage } from '@/shared/lib/api/pages'
import { useTreeStore } from '@/features/tree/state/tree'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { PinnedPageItem } from '@/features/tree/PinnedPageItem'

export function PinnedSection() {
  const { t } = useTranslation('viewer')
  const pinnedPages = useTreeStore((s) => s.pinnedPages)
  const setPinnedLocally = useTreeStore((s) => s.setPinnedLocally)

  // 没有固定项时不渲染空区块。
  if (pinnedPages.length === 0) return null

  const handleUnpin = async (id: string, version: string) => {
    // 先等待接口返回新版本，再修改本地树和固定列表；失败时保留原状态并提示错误。
    try {
      const updated = await pinPage(id, version, false)
      setPinnedLocally(id, false, updated.version)
      toast.success(t('pinned.unpinSuccess'))
    } catch (err) {
      toast.error(mapApiError(err, t('pinned.pinError')).message)
    }
  }

  return (
    <div className="tree-view__pinned" data-testid="pinned-section">
      {pinnedPages.map((node) => (
        <PinnedPageItem
          key={node.id}
          node={node}
          onUnpin={() => handleUnpin(node.id, node.version)}
        />
      ))}
    </div>
  )
}
