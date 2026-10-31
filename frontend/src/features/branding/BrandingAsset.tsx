// 展示品牌图片并记录加载失败的完整地址，自定义图片失败后使用同种类的内置资源。
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { brandingAssetUrl } from './assets'

type BrandingAssetProps = {
  // 文件名与版本共同决定请求地址；替代文本和样式直接传给图片元素。
  kind: 'logo' | 'favicon'
  file: string
  version: number
  alt: string
  className?: string
}

// 失败列表与组件实例一起保留，文件名或版本变化后的地址不受旧地址失败记录影响。
export function BrandingAsset({
  kind,
  file,
  version,
  alt,
  className,
}: BrandingAssetProps) {
  const { t } = useTranslation('branding')
  const [failedUrls, setFailedUrls] = useState<string[]>([])
  const customUrl = brandingAssetUrl(kind, file, version)
  const builtinUrl = brandingAssetUrl(kind, '', version)
  // 优先选择尚未失败的资源地址，回退地址也失败时改为显示状态文案。
  const source =
    customUrl && !failedUrls.includes(customUrl) ? customUrl : builtinUrl
  const unavailable = source !== null && failedUrls.includes(source)

  return (
    <>
      {unavailable ? (
        <span role="status">{t(`${kind}.unavailable`)}</span>
      ) : (
        <img
          src={source ?? undefined}
          alt={alt}
          className={className}
          onError={() => {
            // 将实际失败地址加入列表，由下一次渲染完成回退或显示不可用状态。
            if (source) setFailedUrls((urls) => [...urls, source])
          }}
        />
      )}
    </>
  )
}
