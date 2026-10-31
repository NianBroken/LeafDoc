// 通用分页条使用从零开始的页索引，展示时转换为从一开始的页码。
import { useTranslation } from 'react-i18next'

// Pagination 根据总条数和每页大小计算页数，只有一页或更少时不显示。
// 参数校验由调用方负责，limit 须为正数，page 应处于有效范围。
export function Pagination({
  total,
  page,
  limit,
  onPageChange,
}: {
  total: number
  page: number
  limit: number
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation('common')
  // 上下页回调分别约束下界和上界，不在组件内保存页索引或读取数据。
  const totalPages = Math.ceil(total / limit)
  if (totalPages <= 1) return null

  return (
    <div className="pagination">
      <button
        onClick={() => onPageChange(Math.max(0, page - 1))}
        disabled={page === 0}
        className="pagination__button"
      >
        {t('pagination.prev')}
      </button>
      <span>
        {t('pagination.pageOf', { page: page + 1, total: totalPages })}
      </span>
      <button
        onClick={() => onPageChange(Math.min(totalPages - 1, page + 1))}
        disabled={page >= totalPages - 1}
        className="pagination__button"
      >
        {t('pagination.next')}
      </button>
    </div>
  )
}
