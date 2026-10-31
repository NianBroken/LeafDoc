// 使用图片错误事件与属性断言检查预览回退，不执行浏览器网络请求。
import '@/shared/lib/i18n'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { BrandingAsset } from './BrandingAsset'

// 每个用例通过种类、文件名和版本控制资源地址，必要时重新渲染同一组件观察状态保留。
describe('BrandingAsset', () => {
  it('shows the built-in logo when no custom file exists', () => {
    render(<BrandingAsset kind="logo" file="" version={1} alt="Logo" />)
    expect(screen.getByRole('img', { name: 'Logo' })).toHaveAttribute(
      'src',
      '/defaults/leafdoc-logo.webp?v=1',
    )
    expect(screen.queryByText('Built-in logo')).not.toBeInTheDocument()
  })

  it('falls back from a broken logo and retries when its version changes', () => {
    // 旧版本失败后应显示内置图片，版本变化后应再次使用自定义地址。
    const { rerender } = render(
      <BrandingAsset kind="logo" file="logo.png" version={1} alt="Logo" />,
    )
    fireEvent.error(screen.getByRole('img'))
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      '/defaults/leafdoc-logo.webp?v=1',
    )
    rerender(
      <BrandingAsset kind="logo" file="logo.png" version={2} alt="Logo" />,
    )
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      '/branding/logo.png?v=2',
    )
  })

  it('previews the favicon actually used after custom deletion', () => {
    // 删除操作在组件输入上体现为空文件名，预览因此选择内置图标地址。
    const { rerender } = render(
      <BrandingAsset
        kind="favicon"
        file="favicon.png"
        version={1}
        alt="Icon"
      />,
    )
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      '/branding/favicon.png?v=1',
    )
    rerender(<BrandingAsset kind="favicon" file="" version={2} alt="Icon" />)
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      '/defaults/leafdoc-favicon.ico?v=2',
    )
    expect(screen.queryByText('Built-in favicon')).not.toBeInTheDocument()
  })

  it('reports absence only after both favicon sources fail', () => {
    // 连续派发两次错误，分别使自定义地址与内置地址失败，再断言状态提示。
    render(
      <BrandingAsset
        kind="favicon"
        file="favicon.png"
        version={1}
        alt="Icon"
      />,
    )
    fireEvent.error(screen.getByRole('img'))
    expect(screen.getByRole('img')).toHaveAttribute(
      'src',
      '/defaults/leafdoc-favicon.ico?v=1',
    )
    fireEvent.error(screen.getByRole('img'))
    expect(screen.getByRole('status')).toHaveTextContent('No available favicon')
  })
})
