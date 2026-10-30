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

// Both labels share intrinsic sizing so translations keep preview centers aligned.
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
  const saveInFlight = useRef(false)

  const logoInputRef = useRef<HTMLInputElement>(null)
  const faviconInputRef = useRef<HTMLInputElement>(null)

  useLayoutEffect(() => {
    loadBranding()
  }, [loadBranding])

  useEffect(() => {
    setLocalSiteName(siteName)
  }, [siteName])

  const handleLogoDelete = async () => {
    try {
      await deleteLogo()
      toast.success(t('toast.logoDeleted'))
    } catch (err) {
      toast.error(mapApiError(err, t('toast.logoDeleteFailed')).message)
    }
  }

  const handleFaviconDelete = async () => {
    try {
      await deleteFavicon()
      toast.success(t('toast.faviconDeleted'))
    } catch (err) {
      toast.error(mapApiError(err, t('toast.faviconDeleteFailed')).message)
    }
  }

  const handleSave = async () => {
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
      saveInFlight.current = false
      setSaving(false)
    }
  }

  const handleLogoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
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
