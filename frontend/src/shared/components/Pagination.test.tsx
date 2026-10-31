// 分页测试检查可见文字与点击回调，不执行数据查询，页索引仍由父组件提供。
import '@/shared/lib/i18n'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Pagination } from '@/shared/components/Pagination'

describe('Pagination', () => {
  it('renders nothing when there is only a single page', () => {
    // 总条数小于每页大小，计算页数为一，整个分页条不挂载。
    const { container } = render(
      <Pagination total={5} page={0} limit={10} onPageChange={vi.fn()} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('renders translated prev/next labels and the interpolated page counter', () => {
    // 测试环境的英语资源用于可见文字断言，三十条按十条分成三页。
    render(<Pagination total={30} page={1} limit={10} onPageChange={vi.fn()} />)
    expect(screen.getByText('← Prev')).toBeInTheDocument()
    expect(screen.getByText('Next →')).toBeInTheDocument()
    // 内部索引一对应显示第二页，插值后的总页数应为三。
    expect(screen.getByText('Page 2 of 3')).toBeInTheDocument()
  })

  it('clamps navigation to the valid page range', async () => {
    // 初始页的上一页按钮禁用，下一页点击只报告索引一，不自行改变传入 page。
    const onPageChange = vi.fn()
    render(
      <Pagination total={30} page={0} limit={10} onPageChange={onPageChange} />,
    )
    const user = userEvent.setup()

    await user.click(screen.getByText('← Prev'))
    expect(onPageChange).not.toHaveBeenCalled() // 禁用的首页上一页按钮不触发回调。

    await user.click(screen.getByText('Next →'))
    expect(onPageChange).toHaveBeenCalledWith(1)
  })
})
