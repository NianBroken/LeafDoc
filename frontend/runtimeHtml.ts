// 开发与预览服务为页面请求读取后端站点信息，并在返回 HTML 前填充模板占位符。
import fs from 'node:fs/promises'
import path from 'node:path'
import type { Plugin, PreviewServer, ViteDevServer } from 'vite'

const REQUEST_TIMEOUT_MS = 10000
const HTML_TYPE = 'text/html; charset=utf-8'
// 同时用于属性值与站点标题，替换五种 HTML 特殊字符，不解析输入中的标签。
const escapeAttribute = (value: string) =>
  value.replace(
    /[&<>"']/g,
    (character) =>
      ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[
        character
      ]!,
  )

// 插件只注册开发与预览服务钩子，生产构建保留占位符供 Go 服务填充。
// backendTarget 决定站点信息接口的来源，页面响应不使用本地缓存的站点信息。
export function runtimeHtml(backendTarget: string): Plugin {
  // 每次渲染发起独立请求，超时、非成功状态和缺少有效站点名均进入页面失败分支。
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
    // 文件名作为单个 URL 片段编码，未提供图标文件时使用根目录默认图标地址。
    const favicon = branding.faviconFile
      ? `/branding/${encodeURIComponent(branding.faviconFile)}`
      : '/favicon.ico'
    // 替换函数直接返回转义后的文本，站点名中的美元符号不会被当作替换模式解释。
    // 开发和预览使用根路径；完整站点资料以 JSON 写入 meta，供前端初始化读取。
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
      // 仅接管页面 GET 请求，源码、接口、附件和带其他扩展名的资源继续交给后续中间件。
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
        // 开发读取源码模板，预览读取构建目录模板；每次读取均使用 UTF-8。
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
        // 预览把双引号开头的相对静态地址转为根路径，避免深层页面路径影响资源定位。
        else html = html.replaceAll('"./static/', '"/static/')
        response.writeHead(200, {
          'Content-Type': HTML_TYPE,
          'Cache-Control': 'no-store',
        })
        response.end(html)
      } catch (error) {
        // 页面组装失败时记录实际原因并返回 503，不以未填充模板充当成功页面。
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
