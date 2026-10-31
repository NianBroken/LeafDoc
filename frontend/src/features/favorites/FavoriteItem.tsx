// 显示收藏节点的阅读入口与取消收藏按钮，节点标题和路径来自页面树。
import { FavoriteToggleButton } from '@/features/favorites/FavoriteToggleButton'
import { NODE_KIND_SECTION, PageNode } from '@/shared/lib/api/pages'
import { buildViewUrl } from '@/shared/lib/routePath'
import { useTreeStore } from '@/features/tree/state/tree'
import clsx from 'clsx'
import { File, Folder } from 'lucide-react'
import { Link } from 'react-router'

type Props = {
  node: PageNode
}

export function FavoriteItem({ node }: Props) {
  const activeNodeId = useTreeStore((s) => s.activeNodeId)
  // 选中样式按节点 ID 判断，目录节点使用文件夹图标，其余节点使用文件图标。
  const isActive = activeNodeId === node.id
  const Icon = node.kind === NODE_KIND_SECTION ? Folder : File

  return (
    <div
      className={clsx('tree-view__favorite-item', {
        'tree-view__favorite-item--active': isActive,
      })}
      data-testid="favorite-item"
    >
      <Link
        // 由阅读路由工具转换节点路径，收藏操作按钮位于链接外侧。
        to={buildViewUrl(`/${node.path}`)}
        className="tree-view__favorite-item-link"
      >
        <Icon size={13} className="tree-view__favorite-item-icon" />
        <span className="tree-view__favorite-item-title">{node.title}</span>
      </Link>
      <FavoriteToggleButton
        pageId={node.id}
        className="tree-view__favorite-item-remove"
      />
    </div>
  )
}
