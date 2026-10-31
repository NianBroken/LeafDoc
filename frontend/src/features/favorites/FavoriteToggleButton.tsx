// 根据共享收藏集合显示星标状态，点击时调用对应动作，并显示接口完成或失败提示。
import { mapApiError } from '@/shared/lib/api/errors'
import { TooltipWrapper } from '@/shared/components/TooltipWrapper'
import { useFavoritesStore } from '@/features/favorites/state/favorites'
import clsx from 'clsx'
import { Star } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

type Props = {
  pageId: string
  size?: number
  className?: string
}

export function FavoriteToggleButton({ pageId, size = 13, className }: Props) {
  const { t } = useTranslation('viewer')
  const isFavorited = useFavoritesStore((s) => s.favoritePageIds.has(pageId))
  const addFavorite = useFavoritesStore((s) => s.addFavorite)
  const removeFavorite = useFavoritesStore((s) => s.removeFavorite)

  const handleToggle = async () => {
    // 分支使用当前渲染读取的收藏状态；状态动作负责提前更新界面及失败回滚。
    try {
      if (isFavorited) {
        await removeFavorite(pageId)
        toast.success(t('favorites.removeSuccess'))
      } else {
        await addFavorite(pageId)
        toast.success(t('favorites.addSuccess'))
      }
    } catch (err) {
      toast.error(mapApiError(err, t('favorites.favoriteError')).message)
    }
  }

  const label = isFavorited
    ? t('favorites.removeFavorite')
    : t('favorites.addFavorite')

  return (
    <TooltipWrapper label={label} side="top" align="start">
      <button
        type="button"
        className={clsx('favorite-toggle-button', className, {
          'favorite-toggle-button--active': isFavorited,
        })}
        onClick={(e) => {
          // 阻止默认动作和冒泡，避免收藏按钮触发所在列表项的导航或选择处理。
          e.preventDefault()
          e.stopPropagation()
          handleToggle()
        }}
        aria-label={label}
        aria-pressed={isFavorited}
        data-testid={`favorite-toggle-${pageId}`}
      >
        <Star size={size} fill={isFavorited ? 'currentColor' : 'none'} />
      </button>
    </TooltipWrapper>
  )
}
