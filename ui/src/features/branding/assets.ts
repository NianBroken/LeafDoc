import { withBasePath } from '@/shared/lib/routePath'

// All branding surfaces resolve custom files and built-in assets through this rule.
export const BUILTIN_LOGO = '🌿'

export function brandingAssetUrl(
  kind: 'logo' | 'favicon',
  file: string,
  version: number,
): string | null {
  const path = file
    ? withBasePath(`/branding/${file}`)
    : kind === 'favicon'
      ? withBasePath('/favicon.svg')
      : null
  return path ? `${path}?v=${version}` : null
}
