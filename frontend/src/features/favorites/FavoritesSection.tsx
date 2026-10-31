// 将当前用户的收藏 ID 与页面树索引连接，生成侧栏中的收藏入口。
import { PageNode } from '@/shared/lib/api/pages'
import { useFavoritesStore } from '@/features/favorites/state/favorites'
import { useTreeStore } from '@/features/tree/state/tree'
import { FavoriteItem } from '@/features/favorites/FavoriteItem'

export function FavoritesSection() {
  const favoritePageIds = useFavoritesStore((s) => s.favoritePageIds)
  const byId = useTreeStore((s) => s.byId)

  const favoritePages = Array.from(favoritePageIds)
    // 未解析到的节点不展示，剩余节点按标题的本地化字符串比较结果排序。
    .map((id) => byId[id])
    .filter((n): n is PageNode => !!n)
    .sort((a, b) => a.title.localeCompare(b.title))

  // 没有可展示节点时不保留空容器，收藏 ID 本身不会因此从共享状态删除。
  if (favoritePages.length === 0) return null

  return (
    <div className="tree-view__favorites" data-testid="favorites-section">
      {favoritePages.map((node) => (
        <FavoriteItem key={node.id} node={node} />
      ))}
    </div>
  )
}
