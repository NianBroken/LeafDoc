// 检查名称表单与图片即时上传的调用边界，状态容器动作由替身记录参数。
import '@/shared/lib/i18n'
import i18next from '@/shared/lib/i18n'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BrandingSettings from './BrandingSettings'
import { useBrandingStore } from './state/branding'
vi.mock('@/features/viewer/setTitle', () => ({ useSetTitle: vi.fn() }))
const updateBranding = vi.fn()
const uploadLogo = vi.fn()
beforeEach(async () => {
  // 固定中文标签并提供已加载状态，保证通过可访问名称找到的控件对应真实设置界面。
  await i18next.changeLanguage('zh')
  vi.clearAllMocks()
  useBrandingStore.setState({
    siteName: 'LeafDoc',
    logoFile: '',
    faviconFile: '',
    isLoaded: true,
    isLoading: false,
    error: null,
    logoExts: ['.png'],
    faviconExts: ['.png'],
    loadBranding: vi.fn().mockResolvedValue(undefined),
    updateBranding,
    uploadLogo,
  })
  updateBranding.mockResolvedValue(undefined)
  uploadLogo.mockResolvedValue(undefined)
})
describe('BrandingSettings saving scope', () => {
  it('submits only the name from its own form', async () => {
    // 保存按钮与名称输入属于同一表单，提交仅调用名称更新，不触发图片上传。
    render(<BrandingSettings />)
    const input = screen.getByRole('textbox', { name: '站点名称' })
    const button = screen.getByRole('button', { name: '保存站点名称' })
    expect(button.closest('form')).toBe(input.closest('form'))
    expect(button.closest('form')).not.toHaveTextContent('上传图片')
    fireEvent.change(input, { target: { value: 'New name' } })
    fireEvent.submit(button.closest('form')!)
    await waitFor(() =>
      expect(updateBranding).toHaveBeenCalledWith({ siteName: 'New name' }),
    )
    expect(uploadLogo).not.toHaveBeenCalled()
  })
  it('retains unsaved names during logo upload', async () => {
    // 先输入未保存名称再选择图片，上传动作应保留本地输入且不调用名称更新。
    render(<BrandingSettings />)
    const input = screen.getByRole('textbox', { name: '站点名称' })
    fireEvent.change(input, { target: { value: 'Draft' } })
    const file = new File(['image'], 'test.png', { type: 'image/png' })
    fireEvent.change(document.querySelector('input[type="file"]')!, {
      target: { files: [file] },
    })
    await waitFor(() => expect(uploadLogo).toHaveBeenCalledWith(file))
    expect(input).toHaveValue('Draft')
    expect(updateBranding).not.toHaveBeenCalled()
  })
})
