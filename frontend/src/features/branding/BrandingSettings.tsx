// 管理站点名称、标志和图标，名称通过独立表单保存，资源选择与删除直接调用状态动作。
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { mapApiError } from '@/shared/lib/api/errors'
import { BrandingAsset } from './BrandingAsset'
import { useBrandingStore } from '@/features/branding/state/branding'
import {
  ImageIcon,
  Loader2,
  SaveIcon,
  TrashIcon,
  UploadIcon,
} from 'lucide-react'
import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useSetTitle } from '@/features/viewer/setTitle'

// 每个预览标签同时渲染两种文字，由样式控制展示布局，非当前种类对辅助阅读隐藏。
function BrandingPreviewLabel({ kind }: { kind: 'logo' | 'favicon' }) {
  const { t } = useTranslation('branding')

  return (
    <span className="settings__preview-label settings__branding-label">
      {(['logo', 'favicon'] as const).map((asset) => (
        <span key={asset} aria-hidden={asset !== kind}>
          {t(`${asset}.currentLabel`)}
        </span>
      ))}
    </span>
  )
}

export default function BrandingSettings() {
  // 状态容器提供服务端配置、文件限制和请求状态，组件保留尚未提交的名称草稿。
  const {
    siteName,
    logoFile,
    faviconFile,
    logoVersion,
    faviconVersion,
    isLoading,
    isLoaded,
    error,
    logoExts,
    maxLogoSize,
    faviconExts,
    maxFaviconSize,
    loadBranding,
    updateBranding,
    uploadLogo,
    uploadFavicon,
    deleteLogo,
    deleteFavicon,
  } = useBrandingStore()

  const { t } = useTranslation('branding')

  useSetTitle({ title: t('pageTitle') })

  const [localSiteName, setLocalSiteName] = useState(siteName)
  const [saving, setSaving] = useState(false)
  // 引用同步阻止重复保存，saving 状态用于控制按钮和输入框的显示与禁用。
  const saveInFlight = useRef(false)

  const logoInputRef = useRef<HTMLInputElement>(null)
  const faviconInputRef = useRef<HTMLInputElement>(null)

  useLayoutEffect(() => {
    // 挂载时请求品牌配置，具体加载失败状态由状态容器保存。
    loadBranding()
  }, [loadBranding])

  useEffect(() => {
    // 仅在共享站点名称变化时重设草稿，单独上传图片不会直接重设名称。
    setLocalSiteName(siteName)
  }, [siteName])

  const handleLogoDelete = async () => {
    // 等待删除动作完成后显示成功提示，失败消息通过统一接口错误映射生成。
    try {
      await deleteLogo()
      toast.success(t('toast.logoDeleted'))
    } catch (err) {
      toast.error(mapApiError(err, t('toast.logoDeleteFailed')).message)
    }
  }

  const handleFaviconDelete = async () => {
    // 图标删除还会由状态动作切换页面图标，这里负责用户操作反馈。
    try {
      await deleteFavicon()
      toast.success(t('toast.faviconDeleted'))
    } catch (err) {
      toast.error(mapApiError(err, t('toast.faviconDeleteFailed')).message)
    }
  }

  const handleSave = async () => {
    // 保存中或任一品牌请求处于加载状态时忽略提交，提交内容只包含站点名称。
    if (saveInFlight.current || isLoading) return
    saveInFlight.current = true
    setSaving(true)
    try {
      await updateBranding({
        siteName: localSiteName,
      })
      toast.success(t('toast.brandingSaved'))
    } catch (err) {
      toast.error(mapApiError(err, t('toast.brandingSaveFailed')).message)
    } finally {
      // 请求无论成功或失败均释放同步锁，并恢复保存按钮状态。
      saveInFlight.current = false
      setSaving(false)
    }
  }

  const handleLogoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    // 仅使用选择列表的第一个文件，立即清空输入值，使同一文件能够再次触发选择事件。
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return

    try {
      await uploadLogo(file)
      toast.success(t('toast.logoUploaded'))
    } catch (err) {
      toast.error(mapApiError(err, t('toast.logoUploadFailed')).message)
    }
  }

  const handleFaviconUpload = async (
    e: React.ChangeEvent<HTMLInputElement>,
  ) => {
    // 未选择文件时直接返回；实际类型和大小检查位于上传接口的服务端实现。
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return

    try {
      await uploadFavicon(file)
      toast.success(t('toast.faviconUploaded'))
    } catch (err) {
      toast.error(mapApiError(err, t('toast.faviconUploadFailed')).message)
    }
  }

  return (
    <>
      <div className="settings">
        <h1 className="settings__title">{t('pageTitle')}</h1>
        {error && (
          <p role="alert" className="settings__notice">
            {error}
          </p>
        )}
        <form
          // 阻止浏览器原生提交刷新页面，名称保存由异步动作处理。
          className="settings__section"
          onSubmit={(event) => {
            event.preventDefault()
            void handleSave()
          }}
        >
          <h2 id="site-name-heading" className="settings__section-title">
            {t('siteName.title')}
          </h2>
          <p className="settings__section-description">
            {t('siteName.description')}
          </p>
          <div className="settings__field">
            <Input
              id="siteName"
              aria-labelledby="site-name-heading"
              value={localSiteName}
              disabled={saving || !isLoaded}
              onChange={(e) => setLocalSiteName(e.target.value)}
              placeholder={t('siteName.placeholder')}
            />
          </div>
          <div className="settings__actions">
            <Button type="submit" disabled={saving || isLoading}>
              {saving ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <SaveIcon className="mr-2 h-4 w-4" />
              )}
              {t('save')}
            </Button>
          </div>
        </form>

        <div className="settings__section">
          <h2 className="settings__section-title">{t('logo.title')}</h2>
          <p className="settings__section-description">
            {t('logo.description')}
          </p>
          <p className="settings__hint mb-4">{t('assetsSavedImmediately')}</p>

          <div className="settings__preview settings__branding-preview">
            <BrandingPreviewLabel kind="logo" />
            <div className="settings__branding-slot">
              <div className="settings__branding-asset">
                {!isLoaded && isLoading ? (
                  <Loader2
                    className="h-4 w-4 animate-spin"
                    aria-label={t('loading')}
                  />
                ) : (
                  <BrandingAsset
                    kind="logo"
                    file={logoFile}
                    version={logoVersion}
                    alt={t('logo.title')}
                    className="preview-image settings__preview-image"
                  />
                )}
              </div>
            </div>
            {logoFile && (
              <Button
                variant="destructive"
                size="sm"
                onClick={handleLogoDelete}
                disabled={isLoading}
                aria-label={t('logo.deleteLabel')}
              >
                <TrashIcon />
              </Button>
            )}
          </div>

          <div className="settings__field">
            <input
              // 隐藏的文件控件由上传按钮触发，accept 为选择器提示，不替代服务端校验。
              type="file"
              ref={logoInputRef}
              onChange={handleLogoUpload}
              accept={logoExts.map((ext) => `${ext}`).join(',')}
              className="hidden"
            />
            <Button
              variant="outline"
              onClick={() => logoInputRef.current?.click()}
              disabled={isLoading}
            >
              <UploadIcon className="mr-2 h-4 w-4" />
              {t('logo.uploadButton')}
            </Button>
            <p className="settings__hint">
              {t('logo.hint', {
                exts: logoExts.map((ext) => ext.toUpperCase()).join(', '),
                size: (maxLogoSize / (1024 * 1024)).toFixed(1),
              })}
            </p>
          </div>
        </div>

        <div className="settings__section">
          <h2 className="settings__section-title">{t('favicon.title')}</h2>
          <p className="settings__section-description">
            {t('favicon.description')}
          </p>
          <p className="settings__hint mb-4">{t('assetsSavedImmediately')}</p>

          <div className="settings__preview settings__branding-preview">
            <BrandingPreviewLabel kind="favicon" />
            <div className="settings__branding-slot">
              <div className="settings__branding-asset settings__branding-asset--favicon">
                {!isLoaded && isLoading ? (
                  <Loader2
                    className="h-4 w-4 animate-spin"
                    aria-label={t('loading')}
                  />
                ) : (
                  <BrandingAsset
                    kind="favicon"
                    file={faviconFile}
                    version={faviconVersion}
                    alt={t('favicon.title')}
                    className="preview-image settings__preview-favicon"
                  />
                )}
              </div>
            </div>
            {faviconFile && (
              <Button
                variant="destructive"
                size="sm"
                onClick={handleFaviconDelete}
                disabled={isLoading}
                aria-label={t('favicon.deleteLabel')}
              >
                <TrashIcon />
              </Button>
            )}
          </div>

          <div className="settings__field">
            <input
              type="file"
              ref={faviconInputRef}
              onChange={handleFaviconUpload}
              accept={faviconExts.map((ext) => `${ext}`).join(',')}
              className="hidden"
            />
            <Button
              variant="outline"
              onClick={() => faviconInputRef.current?.click()}
              disabled={isLoading}
            >
              <ImageIcon className="mr-2 h-4 w-4" />
              {t('favicon.uploadButton')}
            </Button>
            <p className="settings__hint">
              {t('favicon.hint', {
                exts: faviconExts.map((ext) => ext.toUpperCase()).join(', '),
                size: (maxFaviconSize / (1024 * 1024)).toFixed(1),
              })}
            </p>
          </div>
        </div>
      </div>
    </>
  )
}
