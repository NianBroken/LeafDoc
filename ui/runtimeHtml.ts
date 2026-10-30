// Development and preview responses resolve branding before delivering HTML.
import fs from 'node:fs/promises'
import path from 'node:path'
import type { Plugin, PreviewServer, ViteDevServer } from 'vite'

const REQUEST_TIMEOUT_MS = 10000
const HTML_TYPE = 'text/html; charset=utf-8'
const escapeAttribute = (value: string) =>
  value.replace(
    /[&<>"']/g,
    (character) =>
      ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[
        character
      ]!,
  )

// Build output remains a backend template; runtime servers resolve every response.
export function runtimeHtml(backendTarget: string): Plugin {
  async function render(template: string): Promise<string> {
    const response = await fetch(new URL('/api/branding', backendTarget), {
      signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
    })
    if (!response.ok)
      throw new Error(`Branding request failed: ${response.status}`)
    const branding = (await response.json()) as {
      siteName: string
      logoFile: string
      faviconFile: string
    }
    if (typeof branding.siteName !== 'string' || !branding.siteName.trim())
      throw new Error('The backend did not supply a site name')
    const favicon = branding.faviconFile
      ? `/branding/${encodeURIComponent(branding.faviconFile)}`
      : '/favicon.ico'
    return template
      .replaceAll('{{__SITE_NAME__}}', () => escapeAttribute(branding.siteName))
      .replaceAll('{{__BASE_PATH__}}', '')
      .replaceAll('{{__FAVICON_HREF__}}', () => escapeAttribute(favicon))
      .replace(
        '</head>',
        () =>
          `<meta name="leafdoc-branding" content="${escapeAttribute(JSON.stringify(branding))}"></head>`,
      )
  }

  function install(server: ViteDevServer | PreviewServer, preview: boolean) {
    server.middlewares.use(async (request, response, next) => {
      const pathname = new URL(request.url ?? '/', 'http://localhost').pathname
      if (
        request.method !== 'GET' ||
        pathname.startsWith('/@') ||
        (/\.[^/]+$/.test(pathname) && !pathname.endsWith('.html')) ||
        /^\/(api|assets|avatars|branding|defaults|static|src|node_modules)(\/|$)/.test(
          pathname,
        )
      )
        return next()
      try {
        const file = preview
          ? path.resolve(
              server.config.root,
              server.config.build.outDir,
              'index.html',
            )
          : path.resolve(server.config.root, 'index.html')
        let html = await render(await fs.readFile(file, 'utf8'))
        if (!preview)
          html = await (server as ViteDevServer).transformIndexHtml(
            pathname,
            html,
          )
        // Built entry assets must resolve from the root on deep preview routes.
        else html = html.replaceAll('"./static/', '"/static/')
        response.writeHead(200, {
          'Content-Type': HTML_TYPE,
          'Cache-Control': 'no-store',
        })
        response.end(html)
      } catch (error) {
        server.config.logger.error(
          `Cannot render site information: ${error instanceof Error ? error.message : String(error)}`,
        )
        response.writeHead(503, {
          'Content-Type': HTML_TYPE,
          'Cache-Control': 'no-store',
        })
        response.end(
          '<!doctype html><html lang="zh"><head><meta charset="utf-8"><title>服务不可用</title></head><body>无法读取站点信息，请检查后端服务。</body></html>',
        )
      }
    })
  }
  return {
    name: 'runtime-site-information',
    configureServer(server) {
      install(server, false)
    },
    configurePreviewServer(server) {
      install(server, true)
    },
  }
}
