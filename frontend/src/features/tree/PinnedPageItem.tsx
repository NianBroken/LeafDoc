// 展示固定在树顶端的节点入口，根据当前节点 ID 标记选中状态并提供悬停取消操作。
import { NODE_KIND_SECTION, PageNode } from '@/shared/lib/api/pages'
import { buildViewUrl } from '@/shared/lib/routePath'
import { useTreeStore } from '@/features/tree/state/tree'
import clsx from 'clsx'
import { File, Folder, PinOff } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'

type Props = {
  node: PageNode
  onUnpin: () => void
}

export function PinnedPageItem({ node, onUnpin }: Props) {
  const { t } = useTranslation('viewer')
  const activeNodeId = useTreeStore((s) => s.activeNodeId)
  const isActive = activeNodeId === node.id
  const [hovered, setHovered] = useState(false)
  // 仅悬停期间渲染取消按钮，目录和页面分别使用文件夹与文件图标。
  const Icon = node.kind === NODE_KIND_SECTION ? Folder : File

  return (
    <div
      className={clsx('tree-view__pinned-item', {
        'tree-view__pinned-item--active': isActive,
      })}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      data-testid="pinned-page-item"
    >
      <Link
        // 点击标题按阅读路由打开节点，取消按钮位于链接之外以单独处理。
        to={buildViewUrl(`/${node.path}`)}
        className="tree-view__pinned-item-link"
      >
        <Icon size={13} className="tree-view__pinned-item-icon" />
        <span className="tree-view__pinned-item-title">{node.title}</span>
      </Link>
      {hovered && (
        <button
          className="tree-view__pinned-item-unpin"
          onClick={(e) => {
            // 防止按钮触发链接默认导航，取消动作由父组件负责调用接口。
            e.preventDefault()
            onUnpin()
          }}
          title={t('pinned.unpinPage')}
          aria-label={t('pinned.unpinPageAriaLabel')}
        >
          <PinOff size={13} />
        </button>
      )}
    </div>
  )
}
