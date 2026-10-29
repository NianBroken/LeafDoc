import { e2eBaseUrl } from '../helpers/servicePorts';

export const e2eBasePath = new URL(e2eBaseUrl).pathname.replace(/\/$/, '');

export function toAppPath(path: string): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;

  if (!e2eBasePath) {
    return normalizedPath;
  }

  return `${e2eBasePath}${normalizedPath}`;
}
