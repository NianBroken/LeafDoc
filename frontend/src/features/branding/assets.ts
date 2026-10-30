import { withBasePath } from '@/shared/lib/routePath'

// Every branding surface uses the same built-in resources and base-path resolution.
const BUILTIN_FILES = {
  logo: 'leafdoc-logo.webp',
  favicon: 'leafdoc-favicon.ico',
}

export function brandingAssetUrl(
  kind: 'logo' | 'favicon',
  file: string,
  version: number,
): string {
  const path = withBasePath(
    file ? `/branding/${file}` : `/defaults/${BUILTIN_FILES[kind]}`,
  )
  return `${path}?v=${version}`
}
