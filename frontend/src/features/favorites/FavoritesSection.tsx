import { PageNode } from '@/shared/lib/api/pages'
import { useFavoritesStore } from '@/features/favorites/state/favorites'
import { useTreeStore } from '@/features/tree/state/tree'
import { FavoriteItem } from '@/features/favorites/FavoriteItem'

export function FavoritesSection() {
  const favoritePageIds = useFavoritesStore((s) => s.favoritePageIds)
  const byId = useTreeStore((s) => s.byId)

  const favoritePages = Array.from(favoritePageIds)
    .map((id) => byId[id])
    .filter((n): n is PageNode => !!n)
    .sort((a, b) => a.title.localeCompare(b.title))

  if (favoritePages.length === 0) return null

  return (
    <div className="tree-view__favorites" data-testid="favorites-section">
      {favoritePages.map((node) => (
        <FavoriteItem key={node.id} node={node} />
      ))}
    </div>
  )
}
