import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { brandingAssetUrl, BUILTIN_LOGO } from './assets'

type BrandingAssetProps = {
  kind: 'logo' | 'favicon'
  file: string
  version: number
  alt: string
  className?: string
  showSource?: boolean
}

// Image failures fall back to the same built-in resource used by the site.
// Failures are tied to the URL so a new upload or refresh can load again.
export function BrandingAsset({
  kind,
  file,
  version,
  alt,
  className,
  showSource = false,
}: BrandingAssetProps) {
  const { t } = useTranslation('branding')
  const [failedUrls, setFailedUrls] = useState<string[]>([])
  const customUrl = brandingAssetUrl(kind, file, version)
  const builtinUrl = brandingAssetUrl(kind, '', version)
  const source =
    customUrl && !failedUrls.includes(customUrl) ? customUrl : builtinUrl
  const builtin = !file || source !== customUrl
  const unavailable = source !== null && failedUrls.includes(source)

  return (
    <>
      {kind === 'logo' && source === null ? (
        <span
          role="img"
          aria-label={alt}
          className={`${className ?? ''} text-xl`}
        >
          {BUILTIN_LOGO}
        </span>
      ) : unavailable ? (
        <span role="status">{t(`${kind}.unavailable`)}</span>
      ) : (
        <img
          src={source ?? undefined}
          alt={alt}
          className={className}
          onError={() => {
            if (source) setFailedUrls((urls) => [...urls, source])
          }}
        />
      )}
      {showSource && builtin && !unavailable && (
        <span className="settings__hint">{t(`${kind}.builtin`)}</span>
      )}
    </>
  )
}
