// 开发与预览共用唯一双协议监听，Vite 中间件及 WebSocket 不另开端口。
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer } from '@httptoolkit/httpolyglot'
import { createServer as createViteServer } from 'vite'
import sirv from 'sirv'
import { createProxyServer } from 'http-proxy-3'
import tls from 'node:tls'

const root = path.dirname(fileURLToPath(import.meta.url))
const ports = JSON.parse(
  fs.readFileSync(
    path.join(root, '../backend/internal/core/serviceports/ports.json'),
    'utf8',
  ),
)
const preview = process.argv.includes('--preview')
const tlsDirectory =
  process.env.LEAFDOC_TLS_DIR ||
  (process.platform === 'win32'
    ? path.join(process.env.ProgramData || 'C:/ProgramData', 'LeafDoc', 'tls')
    : '/etc/leafdoc/tls')
const certificate =
  process.env.LEAFDOC_TLS_CERT || path.join(tlsDirectory, 'server.crt')
const key = process.env.LEAFDOC_TLS_KEY || path.join(tlsDirectory, 'server.key')
// 每个请求单独选择目标协议，HTTPS 使用系统根及显式部署证书验证上游。
const backend = new URL(
  process.env.LEAFDOC_BACKEND_URL || `http://localhost:${ports.backend}`,
)
if (Number(backend.port) !== ports.backend)
  throw new Error('Backend must use its shared fixed port')
const ca = [
  ...tls.getCACertificates('system'),
  fs.readFileSync(process.env.LEAFDOC_TLS_CA || certificate, 'utf8'),
].join('\n')
const proxy = createProxyServer({
  target(request) {
    return {
      host: backend.hostname,
      port: ports.backend,
      protocol: request.socket.encrypted ? 'https:' : 'http:',
      ca,
    }
  },
  changeOrigin: true,
  secure: true,
})
proxy.on('error', (error, request, response) => {
  console.error('Backend proxy failed:', error)
  if ('writeHead' in response && !response.headersSent) response.writeHead(502)
  response.end()
})
const hostIndex = process.argv.indexOf('--host')
const host =
  hostIndex >= 0
    ? process.argv[hostIndex + 1]
    : process.env.LEAFDOC_HOST || '127.0.0.1'
const portIndex = process.argv.indexOf('--port')
if (portIndex >= 0 && Number(process.argv[portIndex + 1]) !== ports.frontend) {
  throw new Error('Frontend must use the shared fixed frontend port')
}
const server = createServer(
  {
    tls: { cert: fs.readFileSync(certificate), key: fs.readFileSync(key) },
  },
  handleRequest,
)
const vite = await createViteServer({
  root,
  server: {
    middlewareMode: { server },
    ws: preview ? false : { server, clientPort: ports.frontend },
  },
  plugins: preview
    ? [
        {
          name: 'preview-static-resources',
          configureServer(instance) {
            const staticFiles = sirv(path.join(root, 'dist'), { single: false })
            instance.middlewares.use((request, response, next) => {
              const pathname = new URL(request.url || '/', 'http://localhost')
                .pathname
              if (/\.[^/]+$/.test(pathname) && !pathname.endsWith('.html')) {
                staticFiles(request, response, next)
              } else next()
            })
          },
        },
      ]
    : [],
})
function handleRequest(request, response) {
  // 协议来自连接本身，客户端提供的转发头不能改变认证 Cookie 模式。
  request.headers['x-forwarded-proto'] = request.socket.encrypted
    ? 'https'
    : 'http'
  delete request.headers['x-forwarded-ssl']
  delete request.headers['front-end-https']
  const pathname = new URL(request.url || '/', 'http://localhost').pathname
  if (/^\/(api|assets|avatars|branding|defaults)(\/|$)/.test(pathname)) {
    proxy.web(request, response)
  } else vite.middlewares(request, response)
}
// 信号与启动失败统一关闭监听及文件监视器，后台部署不使用此开发入口。
let closing = false
async function close() {
  if (closing) return
  closing = true
  await new Promise((resolve) => server.close(resolve))
  proxy.close()
  await vite.close()
}
process.once('SIGINT', () => {
  void close()
})
process.once('SIGTERM', () => {
  void close()
})
try {
  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(ports.frontend, host, resolve)
  })
  console.log(
    `LeafDoc: http://${host}:${ports.frontend}  https://${host}:${ports.frontend}`,
  )
} catch (error) {
  await close()
  throw error
}
